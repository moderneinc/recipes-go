/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package awssdkv2

import (
	"fmt"
	"sync"

	"github.com/moderneinc/recipes-go/recipes/internal/lstutil"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/parser"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// v1 let a client override the session's settings with trailing *aws.Config
// values; v2 takes functional options over the client's own Options instead.
// Only a region override carries across — the rest of v1's Config is configured
// elsewhere in v2 — and the option literal is parsed rather than assembled, so
// its pointer parameter and body come out the way the parser would have made
// them.
const regionOptionSource = `package p

var _ = func(o *%s.Options) { o.Region = %s }
`

const regionPlaceholder = "awsRegionPlaceholder"

var (
	regionOptionMu    sync.Mutex
	regionOptionCache = map[string]java.Expression{}
)

// regionOption builds `func(o *<local>.Options) { o.Region = <region> }`.
func regionOption(serviceLocal string, region java.Expression) (java.Expression, bool) {
	literal, ok := regionOptionLiteral(serviceLocal)
	if !ok {
		return nil, false
	}
	bound, ok := visitor.Init(&placeholderBind{name: regionPlaceholder, value: region}).Visit(literal, nil).(java.Expression)
	if !ok {
		return nil, false
	}
	return bound, true
}

func regionOptionLiteral(serviceLocal string) (java.Expression, bool) {
	regionOptionMu.Lock()
	defer regionOptionMu.Unlock()
	if literal, ok := regionOptionCache[serviceLocal]; ok {
		return literal, literal != nil
	}
	regionOptionCache[serviceLocal] = nil

	cu, err := parser.NewGoParser().Parse("opt.go", fmt.Sprintf(regionOptionSource, serviceLocal, regionPlaceholder))
	if err != nil {
		return nil, false
	}
	lift := visitor.Init(&funcLiteralLift{})
	lift.Visit(cu, nil)
	if lift.literal == nil {
		return nil, false
	}
	regionOptionCache[serviceLocal] = lift.literal
	return lift.literal, true
}

type funcLiteralLift struct {
	visitor.GoVisitor
	literal java.Expression
}

// A function literal in expression position is wrapped, as the parser wraps
// one; Java's side of the tree takes nothing else there.
func (v *funcLiteralLift) VisitStatementExpression(se *golang.StatementExpression, p any) java.J {
	if md, isFunc := se.Statement.(*java.MethodDeclaration); isFunc && v.literal == nil && (md.Name == nil || md.Name.Name == "") {
		v.literal = se
	}
	return v.GoVisitor.VisitStatementExpression(se, p)
}

type placeholderBind struct {
	visitor.GoVisitor
	name  string
	value java.Expression
}

func (v *placeholderBind) VisitIdentifier(id *java.Identifier, p any) java.J {
	if id.Name != v.name {
		return v.GoVisitor.VisitIdentifier(id, p)
	}
	return lstutil.SetExprPrefix(v.value, id.Prefix)
}

// A client takes trailing configs v1 merged over the session's, each setting
// one carried replacing the session's. A generated merge does the same over
// v2's config, which the client is then built from. The configs are literals
// written at the call, or locals holding nothing but such literals and nil.
const mergeConfigHelper = "awsCompatMergeConfig"

// clientConfigScan finds the locals holding nothing but nil or an aws.Config
// literal, read only as a client constructor's trailing config. A name used any
// other way is disqualified.
type clientConfigScan struct {
	visitor.GoVisitor
	scan  *fileScan
	uses  map[string]int
	clean map[string]int
	bad   map[string]bool
}

func (v *clientConfigScan) VisitVariableDeclarations(vd *java.VariableDeclarations, p any) java.J {
	ptr, isPointer := vd.TypeExpr.(*golang.PointerType)
	if isPointer {
		if fa, isField := ptr.Elem.(*java.FieldAccess); isField {
			if name, isAws := qualifiedRef(fa, v.scan.awsPkg); isAws && name == "Config" {
				for _, rp := range vd.Variables {
					if d := rp.Element; d != nil && d.Name != nil {
						v.clean[d.Name.Name]++
						if d.Initializer != nil && !v.literalOrNil(d.Initializer.Element) {
							v.bad[d.Name.Name] = true
						}
					}
				}
			}
		}
	}
	return v.GoVisitor.VisitVariableDeclarations(vd, p)
}

func (v *clientConfigScan) VisitAssignment(a *java.Assignment, p any) java.J {
	if id, isIdent := a.Variable.(*java.Identifier); isIdent {
		if v.clean[id.Name] > 0 || java.HasMarker[golang.ShortVarDecl](a.Markers) && v.literalOrNil(a.Value.Element) {
			v.clean[id.Name]++
			if !v.literalOrNil(a.Value.Element) {
				v.bad[id.Name] = true
			}
		}
	}
	return v.GoVisitor.VisitAssignment(a, p)
}

func (v *clientConfigScan) VisitMethodInvocation(mi *java.MethodInvocation, p any) java.J {
	if _, isConstructor := v.scan.clientConstructorService(mi); isConstructor {
		for _, arg := range realArgs(mi)[1:] {
			if id, isIdent := arg.(*java.Identifier); isIdent {
				v.clean[id.Name]++
			}
		}
	}
	return v.GoVisitor.VisitMethodInvocation(mi, p)
}

func (v *clientConfigScan) VisitIdentifier(id *java.Identifier, p any) java.J {
	v.uses[id.Name]++
	return id
}

// literalOrNil reports whether expr is nil or `&aws.Config{…}`.
func (v *clientConfigScan) literalOrNil(expr java.Expression) bool {
	if id, isIdent := expr.(*java.Identifier); isIdent {
		return id.Name == "nil"
	}
	return v.scan.addressedConfigLiteral(expr)
}

// addressedConfigLiteral reports whether expr is `&aws.Config{…}`.
func (s *fileScan) addressedConfigLiteral(expr java.Expression) bool {
	unary, isUnary := expr.(*golang.Unary)
	if !isUnary || unary.Operator.Element != golang.AddressOf {
		return false
	}
	comp, isComposite := unary.Expression.(*golang.Composite)
	return isComposite && s.awsConfigLiteral(comp)
}

// clientConfigLocals runs the scan over cu.
func (s *fileScan) clientConfigLocals(cu *golang.CompilationUnit) map[string]bool {
	scan := visitor.Init(&clientConfigScan{scan: s, uses: map[string]int{}, clean: map[string]int{}, bad: map[string]bool{}})
	scan.Visit(cu, nil)
	locals := map[string]bool{}
	for name, clean := range scan.clean {
		// Every appearance of the name is a declaration, an assignment or a
		// constructor argument.
		if !scan.bad[name] && clean > 1 && clean == scan.uses[name] {
			locals[name] = true
		}
	}
	return locals
}

// mergedConfig reports whether a trailing config is one the merge takes: a
// literal written at the call, a local holding only such literals or kept as
// v2's config, or a `...*aws.Config` parameter spread.
func (s *fileScan) mergedConfig(expr java.Expression) bool {
	if spread, isSpread := expr.(*golang.Variadic); isSpread && spread.Postfix {
		id, isIdent := spread.Element.(*java.Identifier)
		return isIdent && s.configVariadics[id.Name]
	}
	if id, isIdent := expr.(*java.Identifier); isIdent {
		return s.clientConfigs[id.Name] || s.configKept[id.Name]
	}
	// A NewConfig chain is the literal its setters amount to.
	if options, isChain := s.configChainOptions(expr); isChain {
		_, ok := s.keptChainLiteral(options)
		return ok
	}
	return s.addressedConfigLiteral(expr)
}

// configVariadicScan records the parameters declared `...*aws.Config`.
type configVariadicScan struct {
	visitor.GoVisitor
	scan *fileScan
}

func (v *configVariadicScan) VisitVariableDeclarations(vd *java.VariableDeclarations, p any) java.J {
	if ptr, isPointer := vd.TypeExpr.(*golang.PointerType); isPointer && vd.Varargs != nil {
		if fa, isField := ptr.Elem.(*java.FieldAccess); isField {
			if name, isAws := qualifiedRef(fa, v.scan.awsPkg); isAws && name == "Config" {
				for _, rp := range vd.Variables {
					if d := rp.Element; d != nil && d.Name != nil {
						v.scan.configVariadics[d.Name.Name] = true
					}
				}
			}
		}
	}
	return v.GoVisitor.VisitVariableDeclarations(vd, p)
}

// clientConfigRegion reads the region out of the trailing config a v1 client
// constructor took, whether written as a config literal or built up through
// aws.NewConfig().WithRegion.
func (s *fileScan) clientConfigRegion(expr java.Expression) (java.Expression, bool) {
	// A config literal here sets the client's own region rather than the
	// session's, which is the one option a v2 client takes for it.
	if options, ok := s.sessionArgOptions(expr); ok {
		if len(options) == 1 && options[0].with == "WithRegion" {
			return options[0].value, true
		}
		return nil, false
	}
	mi, isCall := expr.(*java.MethodInvocation)
	if !isCall || mi.Name == nil || mi.Name.Name != "WithRegion" || mi.Select == nil {
		return nil, false
	}
	inner, isCall := mi.Select.Element.(*java.MethodInvocation)
	if !isCall {
		return nil, false
	}
	if helper, isAws := qualifiedCall(inner, s.awsPkg); !isAws || helper != "NewConfig" || len(realArgs(inner)) != 0 {
		return nil, false
	}
	args := realArgs(mi)
	if len(args) != 1 {
		return nil, false
	}
	return args[0], true
}

// convertibleClientConstructor reports whether a v1 client constructor's
// trailing configs carry over. v2 takes functional options, and only a region
// override has one this recipe writes.
func (s *fileScan) convertibleClientConstructor(mi *java.MethodInvocation) bool {
	args := realArgs(mi)
	if len(args) <= 1 {
		return true
	}
	if len(args) == 2 {
		if _, ok := s.clientConfigRegion(args[1]); ok {
			return true
		}
	}
	for _, arg := range args[1:] {
		if !s.mergedConfig(arg) {
			return false
		}
	}
	return true
}

// v1 took the MFA token provider as a field on the session's options; v2 takes
// it on the assume-role credential options, which the loader is handed a
// function to fill in.
const tokenProviderSource = `package p

var _ = func(o *%s.AssumeRoleOptions) { o.TokenProvider = %s }
`

const tokenProviderPlaceholder = "awsTokenProviderPlaceholder"

var (
	tokenProviderMu    sync.Mutex
	tokenProviderCache = map[string]java.Expression{}
)

// tokenProviderLiteral builds `func(o *stscreds.AssumeRoleOptions) { o.TokenProvider = <provider> }`.
func tokenProviderLiteral(stscredsLocal string, provider java.Expression) (java.Expression, bool) {
	literal, ok := tokenProviderTemplate(stscredsLocal)
	if !ok {
		return nil, false
	}
	bound, ok := visitor.Init(&placeholderBind{name: tokenProviderPlaceholder, value: provider}).Visit(literal, nil).(java.Expression)
	if !ok {
		return nil, false
	}
	return bound, true
}

func tokenProviderTemplate(stscredsLocal string) (java.Expression, bool) {
	tokenProviderMu.Lock()
	defer tokenProviderMu.Unlock()
	if literal, ok := tokenProviderCache[stscredsLocal]; ok {
		return literal, literal != nil
	}
	tokenProviderCache[stscredsLocal] = nil

	cu, err := parser.NewGoParser().Parse("token.go", fmt.Sprintf(tokenProviderSource, stscredsLocal, tokenProviderPlaceholder))
	if err != nil {
		return nil, false
	}
	lift := visitor.Init(&funcLiteralLift{})
	lift.Visit(cu, nil)
	if lift.literal == nil {
		return nil, false
	}
	tokenProviderCache[stscredsLocal] = lift.literal
	return lift.literal, true
}

// v1 held the S3 addressing style on the session's config; v2 holds it on the
// client's own options, read here from the local the config's writes were
// hoisted into.
const pathStyleSource = `package p

var _ = func(o *%s.Options) { o.UsePathStyle = %s }
`

var (
	pathStyleMu    sync.Mutex
	pathStyleCache = map[string]java.Expression{}
)

func pathStyleOption(serviceLocal string) (java.Expression, bool) {
	pathStyleMu.Lock()
	defer pathStyleMu.Unlock()
	if literal, ok := pathStyleCache[serviceLocal]; ok {
		return literal, literal != nil
	}
	pathStyleCache[serviceLocal] = nil

	cu, err := parser.NewGoParser().Parse("pathstyle.go", fmt.Sprintf(pathStyleSource, serviceLocal, pathStyleVar))
	if err != nil {
		return nil, false
	}
	lift := visitor.Init(&funcLiteralLift{})
	lift.Visit(cu, nil)
	if lift.literal == nil {
		return nil, false
	}
	pathStyleCache[serviceLocal] = lift.literal
	return lift.literal, true
}
