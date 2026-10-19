/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package awssdkv2

import (
	"github.com/google/uuid"
	"github.com/moderneinc/recipes-go/recipes/internal/lstutil"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// A v1 session was often configured through a local rather than an inline
// literal — `cfg := aws.Config{Region: …}` then `session.NewSession(&cfg)`. v2
// has no counterpart for the local: its own aws.Config is built by the loader,
// and the v1 literal's *string Region does not fit it. So where the local exists
// only to reach the session, its region is read off and the declaration goes.
type configVarScan struct {
	visitor.GoVisitor
	scan *fileScan
}

// configLocal is what the scan learns about one such local: what its literal
// set, what was written to it afterwards and read off it, how often it reached
// a session, and whether anything else touched it at all.
type configLocal struct {
	options []configOption
	// fields are the settings reached on it, read or written.
	fields      []string
	fieldUses   int
	sessionArgs int
	// singleValue counts the session calls whose result is the config alone,
	// which is all a restructured local can stand in for: `session.NewSession`
	// also answers with an error, and only `session.Must` around it consumes
	// that.
	singleValue int
	uses        int
	reassigned  bool
	// chain records that the local was declared from a NewConfig chain.
	chain bool
	// pointerLiteral records that it was declared as an &aws.Config literal,
	// literalKeys the fields that literal set, and credentialsRead that its
	// Credentials were read rather than written.
	pointerLiteral  bool
	literalKeys     []string
	credentialsRead bool
}

func (v *configVarScan) VisitIdentifier(id *java.Identifier, p any) java.J {
	v.scan.configUses[id.Name]++
	return v.GoVisitor.VisitIdentifier(id, p)
}

func (v *configVarScan) VisitAssignment(a *java.Assignment, p any) java.J {
	a = v.GoVisitor.VisitAssignment(a, p).(*java.Assignment)
	switch target := a.Variable.(type) {
	case *java.Identifier:
		if options, ok := v.scan.configLiteralOptions(a.Value.Element); ok {
			local := v.scan.configLocals[target.Name]
			if local == nil {
				local = &configLocal{}
				v.scan.configLocals[target.Name] = local
			}
			local.options = options
			value := a.Value.Element
			if unary, isUnary := value.(*golang.Unary); isUnary {
				value = unary.Expression
			}
			_, local.chain = v.scan.configChainOptions(value)
			local.pointerLiteral, local.literalKeys = false, nil
			if unary, isUnary := a.Value.Element.(*golang.Unary); isUnary && unary.Operator.Element == golang.AddressOf {
				if comp, isComposite := unary.Expression.(*golang.Composite); isComposite {
					local.pointerLiteral = true
					for key := range keyedElements(comp) {
						local.literalKeys = append(local.literalKeys, key)
					}
				}
			}
			v.scan.configVars[target.Name] = options
		} else if local := v.scan.configLocals[target.Name]; local != nil {
			local.reassigned = true
		}
	}
	return a
}

func (v *configVarScan) VisitFieldAccess(fa *java.FieldAccess, p any) java.J {
	if recv, isIdent := fa.Target.(*java.Identifier); isIdent && fa.Name.Element != nil {
		if local := v.scan.configLocals[recv.Name]; local != nil {
			local.fieldUses++
			local.fields = append(local.fields, fa.Name.Element.Name)
			if fa.Name.Element.Name == credentialsField && !assignedTo(v.Cursor()) {
				local.credentialsRead = true
			}
		}
	}
	return v.GoVisitor.VisitFieldAccess(fa, p)
}

func (v *configVarScan) VisitMethodInvocation(mi *java.MethodInvocation, p any) java.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*java.MethodInvocation)
	if v.scan.retryerStatement(mi) {
		name := realArgs(mi)[0].(*java.Identifier).Name
		local := v.scan.configLocals[name]
		retryer, ok := v.scan.configFieldOption(retryerField, realArgs(mi)[1])
		if local == nil || !local.chain || !ok {
			delete(v.scan.retryerStatements, name)
			v.scan.reason = "request.WithRetryer on a config variable not built by aws.NewConfig"
			return mi
		}
		local.options = append(local.options, retryer...)
		v.scan.configVars[name] = local.options
		// The statement is folded away with the local, so its reference is not
		// a use.
		v.scan.configUses[name]--
		return mi
	}
	call, isSession := qualifiedCall(mi, v.scan.sessionPkg)
	if !isSession {
		return mi
	}
	args := realArgs(mi)
	if len(args) != 1 {
		return mi
	}
	switch call {
	case "New", "NewSession", "NewSessionWithOptions":
		if local := v.scan.sessionArgLocal(call, args[0]); local != nil {
			local.sessionArgs++
			if call == "New" {
				local.singleValue++
			}
		}
	case "Must":
		// Must consumes the error, leaving the config alone.
		if inner, isCall := args[0].(*java.MethodInvocation); isCall {
			innerCall, _ := qualifiedCall(inner, v.scan.sessionPkg)
			if innerArgs := realArgs(inner); len(innerArgs) == 1 {
				if local := v.scan.sessionArgLocal(innerCall, innerArgs[0]); local != nil {
					local.singleValue++
				}
			}
		}
	}
	return mi
}

// sessionArgLocal returns the config local a session constructor's argument
// names, which for NewSessionWithOptions is its options' Config.
func (s *fileScan) sessionArgLocal(call string, arg java.Expression) *configLocal {
	if call != "NewSessionWithOptions" {
		return s.localFor(arg)
	}
	comp, isOptions := s.sessionOptionsLiteral(arg)
	if !isOptions {
		return nil
	}
	if config := keyedElements(comp)[sessionConfig]; config != nil {
		return s.localFor(config)
	}
	return nil
}

// localFor returns the config local a session argument names.
func (s *fileScan) localFor(expr java.Expression) *configLocal {
	name, isLocal := s.sessionConfigVar(expr)
	if !isLocal {
		return nil
	}
	return s.configLocals[name]
}

// classifyConfigLocals decides what becomes of each local: the declaration goes
// and its literal folds into the load, or the load moves to the declaration and
// the writes that follow are retargeted onto v2's own config.
func (s *fileScan) classifyConfigLocals() {
	for name, local := range s.configLocals {
		local.uses = s.configUses[name]
		// The declaration and the field reaches account for themselves; a use
		// beyond those and the session calls is something the rewrite does not
		// follow.
		if local.reassigned || local.uses != 1+local.fieldUses+local.sessionArgs {
			// Something else reads it, so it stays v2's config; what it holds is
			// read back as loader options wherever it reaches a session.
			s.configMutated[name] = true
			s.configKept[name] = allMapped(local.fields)
			// A NewConfig chain is kept only where v2's config has a field for
			// each of its settings.
			if local.chain && !local.reassigned {
				if _, ok := s.keptChainLiteral(local.options); !ok {
					s.configKept[name] = false
				}
			}
			continue
		}
		if len(local.fields) == 0 {
			continue
		}
		// The load moves to the declaration only where every session call it
		// feeds wanted the config alone. Otherwise the local stays, holding the
		// settings the generated loader options read back, as long as each is
		// one v2's config holds.
		if local.sessionArgs != local.singleValue {
			s.configKept[name] = allMapped(local.fields)
			s.configMutated[name] = !s.configKept[name]
			continue
		}
		for _, field := range local.fields {
			if _, mapped := configTargets[field]; mapped {
				continue
			}
			if field == s3ForcePathStyle {
				s.configPathStyle[name] = true
				continue
			}
			s.configMutated[name] = true
			break
		}
		if !s.configMutated[name] {
			s.configRestructured[name] = true
		}
	}
}

// allMapped reports whether every setting a config local reaches has a v2
// counterpart on the config itself.
func allMapped(fields []string) bool {
	for _, field := range fields {
		if _, mapped := configTargets[field]; !mapped && field != s3ForcePathStyle {
			return false
		}
	}
	return true
}

// v1 held the S3 addressing style on the session's config; v2 holds it on the
// S3 client's own options, which a different statement builds. A conditional
// write to it therefore cannot follow the config, so it is hoisted into a local
// the client constructor reads instead.
const (
	s3ForcePathStyle = "S3ForcePathStyle"
	pathStyleVar     = "awsUsePathStyle"
)

// sessionConfigVar returns the config local a session argument names, when it
// names one.
func (s *fileScan) sessionConfigVar(expr java.Expression) (string, bool) {
	if unary, isUnary := expr.(*golang.Unary); isUnary && (unary.Operator.Element == golang.AddressOf || unary.Operator.Element == golang.Indirection) {
		expr = unary.Expression
	}
	id, isIdent := expr.(*java.Identifier)
	if !isIdent {
		return "", false
	}
	if _, held := s.configVars[id.Name]; !held {
		return "", false
	}
	return id.Name, true
}

// sessionArgOptions reads the loader options a session constructor's argument
// amounts to, through a local when that is how the config was built.
func (s *fileScan) sessionArgOptions(expr java.Expression) ([]configOption, bool) {
	if options, ok := s.configLiteralOptions(expr); ok {
		return options, true
	}
	name, ok := s.sessionConfigVar(expr)
	if !ok || !s.droppableConfigVar(name) {
		return nil, false
	}
	return s.configVars[name], true
}

// keptConfigVar reports whether a session argument names a config local that
// stays, its settings read back as loader options.
func (s *fileScan) keptConfigVar(expr java.Expression) bool {
	name, isLocal := s.sessionConfigVar(expr)
	return isLocal && s.configKept[name]
}

// restructuredConfigVar returns the config local a session argument names, when
// that local is one whose declaration takes over the load.
func (s *fileScan) restructuredConfigVar(expr java.Expression) (string, bool) {
	name, isLocal := s.sessionConfigVar(expr)
	return name, isLocal && s.configRestructured[name]
}

// droppableConfigVar reports whether the local exists only to reach the session.
// Its two uses are the declaration and the session argument; a third means
// something else reads a config this migration has nothing to put in its place.
func (s *fileScan) droppableConfigVar(name string) bool {
	if s.configMutated[name] {
		return false
	}
	if s.configUses[name] == 2 {
		return true
	}
	// A local making several sessions and nothing else is repeated at each,
	// as long as repeating it evaluates nothing twice.
	local := s.configLocals[name]
	if local == nil || local.reassigned || local.fieldUses > 0 || local.sessionArgs < 2 || s.configUses[name] != 1+local.sessionArgs {
		return false
	}
	for _, o := range local.options {
		if o.with != "WithRetryer" && !sideEffectFree(o.value) {
			return false
		}
	}
	return true
}

// dropsConfigDecl reports whether a statement is the declaration of a config
// local whose every use the migration consumed.
func (v *migrateVisitor) dropsConfigDecl(stmt java.Statement) bool {
	assignment, isAssignment := stmt.(*java.Assignment)
	if !isAssignment || !java.HasMarker[golang.ShortVarDecl](assignment.Markers) {
		return false
	}
	target, isIdent := assignment.Variable.(*java.Identifier)
	if !isIdent {
		return false
	}
	if _, held := v.scan.configVars[target.Name]; !held {
		return false
	}
	return v.scan.droppableConfigVar(target.Name)
}

// referencesQualifier reports whether anything in cu still names pkg as a
// package qualifier. A rewrite that drops a whole statement takes its references
// with it, which the flags set during the visit cannot unlearn.
func referencesQualifier(cu *golang.CompilationUnit, pkg string) bool {
	if pkg == "" {
		return false
	}
	scan := visitor.Init(&qualifierScan{pkg: pkg})
	scan.Visit(cu, nil)
	return scan.found
}

type qualifierScan struct {
	visitor.GoVisitor
	pkg   string
	found bool
}

func (v *qualifierScan) VisitFieldAccess(fa *java.FieldAccess, p any) java.J {
	if target, ok := fa.Target.(*java.Identifier); ok && target.Name == v.pkg {
		v.found = true
	}
	return v.GoVisitor.VisitFieldAccess(fa, p)
}

func (v *qualifierScan) VisitMethodInvocation(mi *java.MethodInvocation, p any) java.J {
	if mi.Select != nil {
		if recv, ok := mi.Select.Element.(*java.Identifier); ok && recv.Name == v.pkg {
			v.found = true
		}
	}
	return v.GoVisitor.VisitMethodInvocation(mi, p)
}

// renamedConfigQualifier rewrites the `config` the loader templates emit to the
// name the file's import actually binds, which differs where the file uses
// `config` for something of its own.
func (v *migrateVisitor) renamedConfigQualifier(tree java.J) java.J {
	if v.scan.configLocal == configPkgName {
		return tree
	}
	renamed, ok := visitor.Init(&qualifierRename{from: configPkgName, to: v.scan.configLocal}).Visit(tree, nil).(java.J)
	if !ok {
		return tree
	}
	return renamed
}

// errorsAlias is the alias an added standard-library errors import needs, or nil
// where the package's own name is free.
func (v *migrateVisitor) errorsAlias() *string {
	if v.scan.errorsLocal == errorsPkg {
		return nil
	}
	alias := v.scan.errorsLocal
	return &alias
}

// renamedErrorsQualifier is renamedConfigQualifier for the errors package.
func (v *migrateVisitor) renamedErrorsQualifier(tree java.J) java.J {
	if v.scan.errorsLocal == errorsPkg {
		return tree
	}
	renamed, ok := visitor.Init(&qualifierRename{from: errorsPkg, to: v.scan.errorsLocal}).Visit(tree, nil).(java.J)
	if !ok {
		return tree
	}
	return renamed
}

type qualifierRename struct {
	visitor.GoVisitor
	from, to string
}

func (v *qualifierRename) VisitMethodInvocation(mi *java.MethodInvocation, p any) java.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*java.MethodInvocation)
	if mi.Select == nil {
		return mi
	}
	recv, ok := mi.Select.Element.(*java.Identifier)
	if !ok || recv.Name != v.from {
		return mi
	}
	c := *mi
	c.Select = &java.RightPadded[java.Expression]{
		Element: &java.Identifier{Prefix: recv.Prefix, Name: v.to, Type: recv.Type},
		After:   mi.Select.After,
	}
	return &c
}

// hoistsPathStyle reports whether a function declares a config local whose S3
// addressing style is carried in a local of its own.
func hoistsPathStyle(md *java.MethodDeclaration, scan *fileScan) bool {
	// Only an S3 client the function builds itself can read the local; a
	// config that leaves the function carries the setting as a service option.
	if !buildsS3Client(md, scan) {
		return false
	}
	for name := range scan.configPathStyle {
		if scan.configRestructured[name] && declaresName(md, name) {
			return true
		}
	}
	return false
}

// buildsS3Client reports whether md constructs an S3 client.
func buildsS3Client(md *java.MethodDeclaration, scan *fileScan) bool {
	check := visitor.Init(&s3ConstructorScan{scan: scan})
	check.Visit(md, nil)
	return check.found
}

type s3ConstructorScan struct {
	visitor.GoVisitor
	scan  *fileScan
	found bool
}

func (v *s3ConstructorScan) VisitMethodInvocation(mi *java.MethodInvocation, p any) java.J {
	if service, isConstructor := v.scan.clientConstructorService(mi); isConstructor && service == s3Service {
		v.found = true
	}
	return v.GoVisitor.VisitMethodInvocation(mi, p)
}

func declaresName(md *java.MethodDeclaration, name string) bool {
	return boundNames(md)[name]
}

// restructuresConfigDecl reports whether a statement declares a config local the
// load moves to, returning the local's name and the options its literal set.
func (v *migrateVisitor) restructuresConfigDecl(stmt java.Statement) (string, []configOption, bool) {
	assignment, isAssignment := stmt.(*java.Assignment)
	if !isAssignment || !java.HasMarker[golang.ShortVarDecl](assignment.Markers) {
		return "", nil, false
	}
	target, isIdent := assignment.Variable.(*java.Identifier)
	if !isIdent || !v.scan.configRestructured[target.Name] {
		return "", nil, false
	}
	if _, isLiteral := v.scan.configLiteralOptions(assignment.Value.Element); !isLiteral {
		return "", nil, false
	}
	return target.Name, v.scan.configVars[target.Name], true
}

// pathStyleWrite reports whether an assignment sets the S3 addressing style on a
// restructured config local, returning the value it sets.
func (v *migrateVisitor) pathStyleWrite(a *java.Assignment) (java.Expression, bool) {
	fa, isField := a.Variable.(*java.FieldAccess)
	if !isField || fa.Name.Element == nil || fa.Name.Element.Name != s3ForcePathStyle {
		return nil, false
	}
	recv, isIdent := fa.Target.(*java.Identifier)
	if !isIdent || !v.scan.configPathStyle[recv.Name] || !v.pathStyleHoisted {
		return nil, false
	}
	value, unwrapped := v.scan.pointerValueSource(a.Value.Element)
	if !unwrapped {
		return nil, false
	}
	return value, true
}

// pathStyleServiceOption rewrites a path-style write to a config the file keeps
// as the generated service option appended to the config's own.
func (v *migrateVisitor) pathStyleServiceOption(a *java.Assignment) (java.J, bool) {
	fa, isField := a.Variable.(*java.FieldAccess)
	if !isField || fa.Name.Element == nil || fa.Name.Element.Name != s3ForcePathStyle {
		return nil, false
	}
	holder, isIdent := fa.Target.(*java.Identifier)
	// A local whose writes are hoisted is the hoisting's to rewrite, and a
	// builder's holds the setting itself.
	if isIdent && (v.scan.configPathStyle[holder.Name] && v.pathStyleHoisted || v.scan.literalBuilderVars[holder.Name]) {
		return nil, false
	}
	kept := isIdent && v.scan.configLocals[holder.Name] != nil
	if !kept && !v.scan.configValue(fa.Target) {
		return nil, false
	}
	force, unwrapped := v.scan.pointerValueSource(a.Value.Element)
	if !unwrapped {
		return nil, false
	}
	options := func() java.Expression {
		return &java.FieldAccess{
			Target: lstutil.SetExprPrefix(fa.Target, java.EmptySpace),
			Name:   java.LeftPadded[*java.Identifier]{Element: &java.Identifier{Name: "ServiceOptions"}},
		}
	}
	v.needsPathStyle = true
	c := *a
	c.Variable = lstutil.SetExprPrefix(options(), a.Variable.GetPrefix())
	c.Value = java.LeftPadded[java.Expression]{Before: a.Value.Before, Element: &java.MethodInvocation{
		Prefix: java.SingleSpace,
		Name:   &java.Identifier{Name: "append"},
		Arguments: java.Container[java.Expression]{Elements: []java.RightPadded[java.Expression]{
			{Element: options()},
			{Element: lstutil.SetExprPrefix(pathStyleCall(force), java.SingleSpace)},
		}},
	}, Markers: a.Value.Markers}
	return &c, true
}

// restructuredConfigDecl rewrites the declaration of a config local as the load
// it now performs, followed where needed by the addressing-style local the v2
// config has no room for.
func (v *migrateVisitor) restructuredConfigDecl(stmt java.Statement, name string, options []configOption) ([]java.Statement, bool) {
	assignment, isAssignment := stmt.(*java.Assignment)
	if !isAssignment {
		return nil, false
	}
	ctx := v.contextExpr()
	if ctx == nil {
		return nil, false
	}
	loaded, built := v.compatConfigCall(ctx, options)
	if !built {
		return nil, false
	}

	declaration := *assignment
	declaration.Value = java.LeftPadded[java.Expression]{
		Before:  assignment.Value.Before,
		Element: lstutil.SetExprPrefix(loaded, java.SingleSpace),
		Markers: assignment.Value.Markers,
	}
	out := []java.Statement{&declaration}
	if !v.scan.configPathStyle[name] || !v.pathStyleHoisted {
		return out, true
	}
	out = append(out, &java.Assignment{
		Prefix:   java.MakeSpace(nil, stmt.GetPrefix().Whitespace()),
		Markers:  java.MakeMarkers(uuid.New(), []java.Marker{golang.ShortVarDecl{Ident: uuid.New()}}),
		Variable: &java.Identifier{Name: pathStyleVar, Type: lstutil.NamedType("bool")},
		Value: java.LeftPadded[java.Expression]{
			Before:  java.SingleSpace,
			Element: &java.Literal{Prefix: java.SingleSpace, Value: false, Source: "false", Type: lstutil.NamedType("bool")},
		},
	})
	return out, true
}
