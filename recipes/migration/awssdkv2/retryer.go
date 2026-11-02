/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package awssdkv2

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/moderneinc/recipes-go/recipes/internal/lstutil"
	"github.com/moderneinc/recipes-go/recipes/migration/internal/pathswap"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/parenthesize"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/parser"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/printer"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// A v1 config's Retryer was usually client.DefaultRetryer tuned through its
// fields, sometimes embedded in a type whose ShouldRetry added a condition of
// its own. It becomes v2's standard retryer: v1's attempts, its larger maximum
// delay as the backoff cap, and no client-side retry budget, which v1 did not
// have. v1's minimum delays have no v2 counterpart. A ShouldRetry adding a
// condition on the request's error becomes a rule v2 consults ahead of its own,
// which retries in the cases v1's `||` did.
const (
	retryerField = "Retryer"
	v2Retry      = v2Module + "/aws/retry"
	v2RateLimit  = v2Module + "/aws/ratelimit"
)

// v1DefaultRetryerConstants are client's DefaultRetryer constants as v2 source.
var v1DefaultRetryerConstants = map[string]string{
	"DefaultRetryerMaxNumRetries":    "3",
	"DefaultRetryerMinRetryDelay":    "30 * time.Millisecond",
	"DefaultRetryerMinThrottleDelay": "500 * time.Millisecond",
	"DefaultRetryerMaxRetryDelay":    "300 * time.Second",
	"DefaultRetryerMaxThrottleDelay": "300 * time.Second",
}

var defaultRetryerFields = map[string]bool{
	"NumMaxRetries": true, "MinRetryDelay": true, "MaxRetryDelay": true,
	"MinThrottleDelay": true, "MaxThrottleDelay": true,
}

// retryerType is a type embedding client.DefaultRetryer whose ShouldRetry adds a
// condition on the request's error.
type retryerType struct {
	decl   *golang.TypeDecl
	method *golang.MethodDeclaration
	// cond is the added condition, and param the request it reads the error of.
	cond  java.Expression
	param string
}

// retryerTypes finds the retryer types cu declares.
func (s *fileScan) retryerTypes(cu *golang.CompilationUnit) map[string]*retryerType {
	types := map[string]*retryerType{}
	if s.clientPkg == "" {
		return types
	}
	for _, rp := range cu.Statements {
		decl, isType := rp.Element.(*golang.TypeDecl)
		if !isType || decl.Name == nil || !s.embedsDefaultRetryerOnly(decl) {
			continue
		}
		types[decl.Name.Name] = &retryerType{decl: decl}
	}
	methods := map[string]int{}
	for _, rp := range cu.Statements {
		md, isMethod := rp.Element.(*golang.MethodDeclaration)
		if !isMethod || md.Declaration == nil || md.Declaration.Name == nil {
			continue
		}
		recv := receiverTypeName(md)
		t := types[recv]
		if t == nil {
			continue
		}
		methods[recv]++
		if md.Declaration.Name.Name != "ShouldRetry" {
			continue
		}
		if cond, param, ok := s.shouldRetryOverride(md.Declaration); ok {
			t.method, t.cond, t.param = md, cond, param
		}
	}
	for name, t := range types {
		// Any method but the override reads v1's retryer in a way v2's has no
		// counterpart for.
		if methods[name] > 1 || (methods[name] == 1 && t.method == nil) {
			delete(types, name)
		}
	}
	return types
}

// embedsDefaultRetryerOnly reports whether decl is `type T struct{ client.DefaultRetryer }`.
func (s *fileScan) embedsDefaultRetryerOnly(decl *golang.TypeDecl) bool {
	st, isStruct := decl.Definition.(*golang.StructType)
	if !isStruct || st.Body == nil || len(st.Body.Statements) != 1 {
		return false
	}
	// An embedded field is a declaration whose one variable has no name.
	vd, isDecl := st.Body.Statements[0].Element.(*java.VariableDeclarations)
	if !isDecl || len(vd.Variables) > 1 {
		return false
	}
	if len(vd.Variables) == 1 && vd.Variables[0].Element != nil && vd.Variables[0].Element.Name != nil && vd.Variables[0].Element.Name.Name != "" {
		return false
	}
	fa, isField := vd.TypeExpr.(*java.FieldAccess)
	if !isField {
		return false
	}
	name, isClient := qualifiedRef(fa, s.clientPkg)
	return isClient && name == "DefaultRetryer"
}

// receiverTypeName names the type a method's receiver binds, through a pointer
// or not.
func receiverTypeName(md *golang.MethodDeclaration) string {
	for _, rp := range md.Receiver.Elements {
		vd, isDecl := rp.Element.(*java.VariableDeclarations)
		if !isDecl {
			continue
		}
		t := vd.TypeExpr
		if ptr, isPointer := t.(*golang.PointerType); isPointer {
			t = ptr.Elem
		}
		if id, isIdent := t.(*java.Identifier); isIdent {
			return id.Name
		}
	}
	return ""
}

// shouldRetryOverride reads `return r.DefaultRetryer.ShouldRetry(req) || cond`,
// where cond reads the request only as req.Error.
func (s *fileScan) shouldRetryOverride(md *java.MethodDeclaration) (java.Expression, string, bool) {
	param, _, isParam := s.requestParam(md)
	if !isParam || md.Body == nil || len(md.Body.Statements) != 1 {
		return nil, "", false
	}
	ret, isReturn := md.Body.Statements[0].Element.(*java.Return)
	if !isReturn || ret.Expression == nil {
		return nil, "", false
	}
	bin, isBinary := ret.Expression.(*java.Binary)
	// A tree read back over RPC carries `||` as Or, which the parser calls LogicalOr.
	if !isBinary || (bin.Operator.Element != java.LogicalOr && bin.Operator.Element != java.Or) {
		return nil, "", false
	}
	call, isCall := bin.Left.(*java.MethodInvocation)
	if !isCall || call.Name == nil || call.Name.Name != "ShouldRetry" || call.Select == nil {
		return nil, "", false
	}
	embedded, isField := call.Select.Element.(*java.FieldAccess)
	if !isField || embedded.Name.Element == nil || embedded.Name.Element.Name != "DefaultRetryer" {
		return nil, "", false
	}
	if args := realArgs(call); len(args) != 1 || exprKey(args[0]) != param {
		return nil, "", false
	}
	scan := visitor.Init(&requestErrorScan{param: param})
	scan.Visit(bin.Right, nil)
	if scan.uses != scan.errorReads {
		return nil, "", false
	}
	return bin.Right, param, true
}

// requestErrorScan counts a request's uses, and those that read its Error.
type requestErrorScan struct {
	visitor.GoVisitor
	param            string
	uses, errorReads int
}

func (v *requestErrorScan) VisitIdentifier(id *java.Identifier, p any) java.J {
	if id.Name == v.param {
		v.uses++
	}
	return id
}

func (v *requestErrorScan) VisitFieldAccess(fa *java.FieldAccess, p any) java.J {
	if exprKey(fa.Target) == v.param && fa.Name.Element != nil && fa.Name.Element.Name == "Error" {
		v.errorReads++
	}
	return v.GoVisitor.VisitFieldAccess(fa, p)
}

// retryerLiteral reads a Retryer value: `client.DefaultRetryer{…}`, or a
// retryer type wrapping one, with or without its address taken.
func (s *fileScan) retryerLiteral(expr java.Expression) (fields map[string]java.Expression, typeName string, ok bool) {
	if unary, isUnary := expr.(*golang.Unary); isUnary && unary.Operator.Element == golang.AddressOf {
		expr = unary.Expression
	}
	comp, isComposite := expr.(*golang.Composite)
	if !isComposite {
		return nil, "", false
	}
	if id, isIdent := comp.TypeExpr.(*java.Identifier); isIdent && s.retryers[id.Name] != nil {
		elements := keyedElements(comp)
		inner := elements["DefaultRetryer"]
		if inner == nil || len(elements) != 1 {
			return nil, "", false
		}
		fields, _, ok = s.retryerLiteral(inner)
		return fields, id.Name, ok
	}
	fa, isField := comp.TypeExpr.(*java.FieldAccess)
	if !isField {
		return nil, "", false
	}
	if name, isClient := qualifiedRef(fa, s.clientPkg); !isClient || name != "DefaultRetryer" {
		return nil, "", false
	}
	fields = keyedElements(comp)
	if fields == nil || fields["NumMaxRetries"] == nil {
		// Without a count, v1 made no retries at all.
		return nil, "", false
	}
	for name := range fields {
		if !defaultRetryerFields[name] {
			return nil, "", false
		}
	}
	return fields, "", true
}

// keyedElements maps a literal's keys to their values, or nil when an element
// is not keyed.
func keyedElements(comp *golang.Composite) map[string]java.Expression {
	out := map[string]java.Expression{}
	for _, rp := range comp.Elements.Elements {
		if _, isEmpty := rp.Element.(*java.Empty); isEmpty {
			continue
		}
		kv, isKeyValue := rp.Element.(*golang.KeyValue)
		if !isKeyValue {
			return nil
		}
		key, isIdent := kv.Key.(*java.Identifier)
		if !isIdent {
			return nil
		}
		out[key.Name] = kv.Value.Element
	}
	return out
}

// retryerSource writes v2's form of a retryer literal, and reports whether it
// names time where the literal did not. A value the literal sets is written as a
// placeholder that values maps to the value's own tree, with any v1 constant in
// it spelled out, as v2 has none of them.
func (s *fileScan) retryerSource(fields map[string]java.Expression, typeName string) (string, map[string]java.Expression, bool) {
	values := map[string]java.Expression{}
	spelledTime := false
	value := func(field, fallback string) (text, source string) {
		expr, set := fields[field]
		if !set {
			return fallback, fallback
		}
		text = s.v1ConstantsAsSource(expr)
		if text != printer.Print(lstutil.SetExprPrefix(expr, java.EmptySpace)) {
			spelling := visitor.Init(&v1ConstantSpelling{scan: s})
			spelled, isExpr := spelling.Visit(expr, nil).(java.Expression)
			if !isExpr || spelling.failed {
				return text, text
			}
			expr, spelledTime = spelled, spelledTime || spelling.namesTime
		}
		values[retryerPlaceholder+field] = expr
		return text, retryerPlaceholder + field
	}
	count, attempts := value("NumMaxRetries", "")
	if n, err := strconv.Atoi(count); err == nil {
		attempts = strconv.Itoa(n + 1)
	} else {
		attempts += " + 1"
	}
	retryText, maxRetry := value("MaxRetryDelay", s.inFileTime(v1DefaultRetryerConstants["DefaultRetryerMaxRetryDelay"]))
	throttleText, maxThrottle := value("MaxThrottleDelay", s.inFileTime(v1DefaultRetryerConstants["DefaultRetryerMaxThrottleDelay"]))
	backoff := maxRetry
	if throttleText != retryText {
		backoff = fmt.Sprintf("max(%s, %s)", maxRetry, maxThrottle)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "func() aws.Retryer {\n\treturn retry.NewStandard(func(o *retry.StandardOptions) {\n")
	fmt.Fprintf(&b, "\t\to.MaxAttempts = %s\n\t\to.MaxBackoff = %s\n\t\to.RateLimiter = ratelimit.None\n", attempts, backoff)
	if typeName != "" {
		fmt.Fprintf(&b, "\t\to.Retryables = append([]retry.IsErrorRetryable{%s{}}, o.Retryables...)\n", typeName)
	}
	b.WriteString("\t})\n}")
	return b.String(), values, spelledTime || strings.Contains(b.String(), s.timeQualifier()+".")
}

// v1ConstantsAsSource prints expr with client's DefaultRetryer constants
// spelled out, as v2 has none of them.
func (s *fileScan) v1ConstantsAsSource(expr java.Expression) string {
	text := printer.Print(lstutil.SetExprPrefix(expr, java.EmptySpace))
	for name, value := range v1DefaultRetryerConstants {
		text = strings.ReplaceAll(text, s.clientPkg+"."+name, s.inFileTime(value))
	}
	return text
}

// v1ConstantSpelling spells out client's DefaultRetryer constants in a value's
// tree, parenthesized where the value binds tighter than the spelling.
type v1ConstantSpelling struct {
	visitor.GoVisitor
	scan *fileScan
	// namesTime records that a spelling names time, and failed that one did not parse.
	namesTime, failed bool
}

func (v *v1ConstantSpelling) VisitFieldAccess(fa *java.FieldAccess, p any) java.J {
	name, isClient := qualifiedRef(fa, v.scan.clientPkg)
	value, isConstant := v1DefaultRetryerConstants[name]
	if !isClient || !isConstant {
		return v.GoVisitor.VisitFieldAccess(fa, p)
	}
	var header string
	if strings.Contains(value, "time.") {
		header = fmt.Sprintf("import %s %q\n\n", v.scan.timeQualifier(), "time")
		v.namesTime = true
	}
	spelled, ok := parsedExpressionAfter(header, v.scan.inFileTime(value))
	if !ok {
		v.failed = true
		return fa
	}
	return parenthesize.Maybe(lstutil.SetExprPrefix(spelled, fa.Prefix), v.Cursor())
}

// inFileTime spells the value of a v1 constant with the name the file calls the
// time package by.
func (s *fileScan) inFileTime(value string) string {
	return strings.ReplaceAll(value, "time.", s.timeQualifier()+".")
}

// timeQualifier is the name the file imports time under, or time where it does
// not import it.
func (s *fileScan) timeQualifier() string {
	if s.timePkg != "" {
		return s.timePkg
	}
	return "time"
}

// retryerPlaceholder begins the name a value of a retryer literal stands under
// in the source of v2's form.
const retryerPlaceholder = "awsRetryer"

// usesTime reports whether a retryer literal's v2 form names the time package
// itself, as it does where it spells out a v1 default or constant. A value the
// literal sets keeps the qualifiers its file wrote it with.
func (s *fileScan) usesTime(fields map[string]java.Expression) bool {
	_, _, namesTime := s.retryerSource(fields, "")
	return namesTime
}

// parsedRetryer parses v2's form of a retryer literal, importing time where it
// names the package, so that a delay such as time.Duration(n) is a conversion,
// and puts back the trees of the values the literal sets.
func (s *fileScan) parsedRetryer(fields map[string]java.Expression, typeName string) (java.Expression, bool) {
	source, values, _ := s.retryerSource(fields, typeName)
	source = strings.ReplaceAll(source, "aws.", s.awsQualifier()+".")
	timeName := s.timeQualifier()
	// Each placeholder is declared with the type of the field it stands for.
	var decls strings.Builder
	for _, field := range []string{"NumMaxRetries", "MaxRetryDelay", "MaxThrottleDelay"} {
		if values[retryerPlaceholder+field] != nil {
			typ := timeName + ".Duration"
			if field == "NumMaxRetries" {
				typ = "int"
			}
			fmt.Fprintf(&decls, "var %s%s %s\n\n", retryerPlaceholder, field, typ)
		}
	}
	header := decls.String()
	if strings.Contains(header+source, timeName+".") {
		header = fmt.Sprintf("import %s %q\n\n", timeName, "time") + header
	}
	parsed, ok := parsedExpressionAfter(header, source)
	if !ok {
		return nil, false
	}
	for _, name := range sortedKeys(values) {
		parsed = visitor.Init(&placeholderBind{name: name, value: values[name]}).Visit(parsed, nil).(java.Expression)
	}
	return parsed, true
}

// parsedExpression parses source standing as an expression, in a file that
// imports the packages at imports, so that a conversion to one of their types
// parses as a conversion rather than as a call.
func parsedExpression(source string, imports ...string) (java.Expression, bool) {
	var header strings.Builder
	for _, path := range imports {
		fmt.Fprintf(&header, "import %q\n\n", path)
	}
	return parsedExpressionAfter(header.String(), source)
}

// parsedExpressionAfter parses source standing as an expression, in a file that
// opens with the imports and declarations of header.
func parsedExpressionAfter(header, source string) (java.Expression, bool) {
	cu, err := parser.NewGoParser().Parse("expr.go", "package p\n\n"+header+"var _ = "+source+"\n")
	if err != nil {
		return nil, false
	}
	for _, rp := range cu.Statements {
		if vd, isDecl := rp.Element.(*java.VariableDeclarations); isDecl && len(vd.Variables) == 1 {
			if init := vd.Variables[0].Element.Initializer; init != nil {
				return init.Element, true
			}
		}
	}
	return nil, false
}

// parsedStatement parses source standing as a top-level declaration.
func parsedStatement(source string) (java.Statement, bool) {
	cu, err := parser.NewGoParser().Parse("decl.go", "package p\n\n"+source+"\n")
	if err != nil || len(cu.Statements) != 1 {
		return nil, false
	}
	return cu.Statements[0].Element, true
}

// retryerValue builds v2's Retryer for a v1 retryer literal.
func (v *migrateVisitor) retryerValue(expr java.Expression) (java.Expression, bool) {
	fields, typeName, ok := v.scan.retryerLiteral(expr)
	if !ok {
		return nil, false
	}
	value, ok := v.scan.parsedRetryer(fields, typeName)
	if !ok {
		return nil, false
	}
	value = v.typedByImports(value).(java.Expression)
	v.needsRetry = true
	v.reformat = true
	v.awsUsed, v.needsAws = true, true
	if v.scan.usesTime(fields) {
		v.needsTime = true
	}
	return value, true
}

// retryerDeclarations rewrites a retryer type and its override as an empty type
// whose method is the rule v2's retryer consults first.
func (v *migrateVisitor) retryerDeclarations(stmt java.Statement) (java.Statement, bool) {
	switch d := stmt.(type) {
	case *golang.TypeDecl:
		t := v.scan.retryers[d.Name.Name]
		if t == nil || t.decl != d {
			return nil, false
		}
		decl, ok := parsedStatement("type " + d.Name.Name + " struct{}")
		if !ok {
			return nil, false
		}
		return lstutil.SetStmtPrefix(decl, d.Prefix), true
	case *golang.MethodDeclaration:
		t := v.scan.retryers[receiverTypeName(d)]
		if t == nil || t.method != d {
			return nil, false
		}
		param := "err"
		if strings.Contains(printer.Print(t.cond), "err") {
			param = "retryErr"
		}
		cond := visitor.Init(&requestErrorSwap{param: t.param, to: param}).Visit(t.cond, nil).(java.Expression)
		// An if takes its condition bare.
		if parens, isParens := cond.(*java.Parentheses); isParens {
			cond = parens.Tree.Element
		}
		recv := strings.TrimSpace(printer.Print(d.Receiver.Elements[0].Element))
		aws := v.scan.awsQualifier()
		source := fmt.Sprintf("func (%s) IsErrorRetryable(%s error) %s.Ternary {\n\tif %s {\n\t\treturn %s.TrueTernary\n\t}\n\treturn %s.UnknownTernary\n}",
			recv, param, aws, strings.TrimSpace(printer.Print(cond)), aws, aws)
		decl, ok := parsedStatement(source)
		if !ok {
			return nil, false
		}
		decl = v.typedByImports(decl).(java.Statement)
		v.awsUsed, v.needsAws = true, true
		return lstutil.SetStmtPrefix(decl, d.Prefix), true
	}
	return nil, false
}

// requestErrorSwap replaces req.Error with the error v2 hands the rule.
type requestErrorSwap struct {
	visitor.GoVisitor
	param, to string
}

func (v *requestErrorSwap) VisitFieldAccess(fa *java.FieldAccess, p any) java.J {
	if exprKey(fa.Target) == v.param && fa.Name.Element != nil && fa.Name.Element.Name == "Error" {
		return &java.Identifier{Prefix: fa.Prefix, Name: v.to, Type: lstutil.NamedType("error")}
	}
	return v.GoVisitor.VisitFieldAccess(fa, p)
}

// retryerRefScan records the client and request references a recognised
// retryer consumes.
type retryerRefScan struct {
	visitor.GoVisitor
	scan *fileScan
}

func (v *retryerRefScan) VisitFieldAccess(fa *java.FieldAccess, p any) java.J {
	if _, isClient := qualifiedRef(fa, v.scan.clientPkg); isClient {
		v.scan.consumedClientRefs[fa] = true
	}
	if _, isRequest := qualifiedRef(fa, v.scan.requestPkg); isRequest {
		v.scan.consumedRequestRefs[fa] = true
	}
	return v.GoVisitor.VisitFieldAccess(fa, p)
}

// consumeRetryers marks the references inside recognised retryer types and
// literals as consumed.
func (s *fileScan) consumeRetryers(cu *golang.CompilationUnit) {
	refs := visitor.Init(&retryerRefScan{scan: s})
	for _, t := range s.retryers {
		refs.Visit(t.decl, nil)
		if t.method != nil {
			refs.Visit(t.method, nil)
		}
	}
	visitor.Init(&retryerLiteralScan{scan: s, refs: refs}).Visit(cu, nil)
}

type retryerLiteralScan struct {
	visitor.GoVisitor
	scan *fileScan
	refs *retryerRefScan
}

// request.WithRetryer hands a retryer to a config as its Retryer field does.
func (v *retryerLiteralScan) VisitMethodInvocation(mi *java.MethodInvocation, p any) java.J {
	if call, isRequest := qualifiedCall(mi, v.scan.requestPkg); isRequest && call == "WithRetryer" && len(realArgs(mi)) == 2 {
		if _, ok := v.scan.configChainOptions(mi); ok {
			retryer := realArgs(mi)[1]
			v.refs.Visit(retryer, nil)
			v.scan.retryerValues[retryer] = true
			v.scan.consumedRequestCalls[mi] = true
		} else if local, isLocal := realArgs(mi)[0].(*java.Identifier); isLocal && isStatement(v.Cursor()) {
			if _, ok := v.scan.configFieldOption(retryerField, realArgs(mi)[1]); ok {
				v.refs.Visit(realArgs(mi)[1], nil)
				v.scan.consumedRequestCalls[mi] = true
				v.scan.retryerStatements[local.Name] = true
			}
		}
	}
	return v.GoVisitor.VisitMethodInvocation(mi, p)
}

// retryerStatement reports whether stmt is `request.WithRetryer(cfg, r)` on a
// local whose options it folded into.
func (s *fileScan) retryerStatement(stmt java.Statement) bool {
	mi, isCall := stmt.(*java.MethodInvocation)
	if !isCall {
		return false
	}
	call, isRequest := qualifiedCall(mi, s.requestPkg)
	if !isRequest || call != "WithRetryer" || len(realArgs(mi)) != 2 {
		return false
	}
	local, isLocal := realArgs(mi)[0].(*java.Identifier)
	return isLocal && s.retryerStatements[local.Name]
}

func (v *retryerLiteralScan) VisitKeyValue(kv *golang.KeyValue, p any) java.J {
	if key, isIdent := kv.Key.(*java.Identifier); isIdent && key.Name == "Retryer" {
		if _, _, ok := v.scan.retryerLiteral(kv.Value.Element); ok {
			v.refs.Visit(kv.Value.Element, nil)
			v.scan.retryerValues[kv.Value.Element] = true
		}
	}
	return v.GoVisitor.VisitKeyValue(kv, p)
}

// rewriteRetryerDeclarations replaces each retryer type and override, matched
// by name since the visit may have rebuilt them.
func (v *migrateVisitor) rewriteRetryerDeclarations(cu *golang.CompilationUnit) *golang.CompilationUnit {
	if len(v.scan.retryers) == 0 {
		return cu
	}
	statements := append([]java.RightPadded[java.Statement](nil), cu.Statements...)
	changed := false
	for i, rp := range statements {
		var target java.Statement
		switch d := rp.Element.(type) {
		case *golang.TypeDecl:
			if t := v.scan.retryers[d.Name.Name]; t != nil {
				target = t.decl
			}
		case *golang.MethodDeclaration:
			if t := v.scan.retryers[receiverTypeName(d)]; t != nil && t.method != nil && d.Declaration != nil && d.Declaration.Name != nil && d.Declaration.Name.Name == "ShouldRetry" {
				target = t.method
			}
		}
		if target == nil {
			continue
		}
		if rewritten, ok := v.retryerDeclarations(target); ok {
			statements[i].Element = rewritten
			changed = true
		}
	}
	if !changed {
		return cu
	}
	c := *cu
	c.Statements = statements
	return &c
}

// qualifiedCallTyping attributes the package-qualified calls in a tree parsed
// apart from the file, from the import path each qualifier names.
type qualifiedCallTyping struct {
	visitor.GoVisitor
	paths map[string]string
}

func (v *qualifiedCallTyping) VisitMethodInvocation(mi *java.MethodInvocation, p any) java.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*java.MethodInvocation)
	recv, isIdent := selectIdentifier(mi)
	if !isIdent || mi.Name == nil {
		return mi
	}
	// A call parsed in its own file, as a value carried over from it is, is
	// already typed with the signature of the package's function.
	path, known := v.paths[recv.Name]
	if !known || declaredIn(mi, path) {
		return mi
	}
	c := *mi
	sel := *mi.Select
	q := *recv
	q.Type = lstutil.NamedType(path)
	sel.Element = &q
	c.Select = &sel
	c.MethodType = lstutil.FuncType(path, mi.Name.Name, nil)
	return &c
}

// declaredIn reports whether mi is typed as a function of the package at path.
func declaredIn(mi *java.MethodInvocation, path string) bool {
	return mi.MethodType != nil && mi.MethodType.DeclaringType != nil &&
		mi.MethodType.DeclaringType.GetFullyQualifiedName() == path
}

// typedByImports attributes tree's qualified calls against the file's imports
// and the packages the retryer rewrite adds.
func (v *migrateVisitor) typedByImports(tree java.J) java.J {
	paths := map[string]string{
		"retry":               v2Retry,
		"ratelimit":           v2RateLimit,
		"time":                "time",
		v.scan.awsQualifier(): v2Aws,
	}
	if cu, isFile := v.Cursor().Value().(*golang.CompilationUnit); isFile && cu.Imports != nil {
		for _, rp := range cu.Imports.Elements {
			if local := pathswap.LocalName(rp.Element); local != "" && paths[local] == "" {
				paths[local] = rp.Element.Path()
			}
		}
	}
	for c := v.Cursor(); c != nil; c = c.Parent() {
		if cu, isFile := c.Value().(*golang.CompilationUnit); isFile && cu.Imports != nil {
			for _, rp := range cu.Imports.Elements {
				if local := pathswap.LocalName(rp.Element); local != "" && paths[local] == "" {
					paths[local] = rp.Element.Path()
				}
			}
		}
	}
	if typed, ok := visitor.Init(&qualifiedCallTyping{paths: paths}).Visit(tree, nil).(java.J); ok {
		return typed
	}
	return tree
}
