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
		}
	}
	return v.GoVisitor.VisitFieldAccess(fa, p)
}

func (v *configVarScan) VisitMethodInvocation(mi *java.MethodInvocation, p any) java.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*java.MethodInvocation)
	call, isSession := qualifiedCall(mi, v.scan.sessionPkg)
	if !isSession {
		return mi
	}
	args := realArgs(mi)
	if len(args) != 1 {
		return mi
	}
	switch call {
	case "New", "NewSession":
		if local := v.scan.localFor(args[0]); local != nil {
			local.sessionArgs++
			if call == "New" {
				local.singleValue++
			}
		}
	case "Must":
		// Must consumes the error, leaving the config alone.
		if inner, isCall := args[0].(*java.MethodInvocation); isCall {
			if innerArgs := realArgs(inner); len(innerArgs) == 1 {
				if local := v.scan.localFor(innerArgs[0]); local != nil {
					local.singleValue++
				}
			}
		}
	}
	return mi
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
			s.configMutated[name] = true
			continue
		}
		if len(local.fields) == 0 {
			continue
		}
		// The load moves to the declaration, so every session call it feeds has
		// to be one that wanted the config alone.
		if local.sessionArgs != local.singleValue {
			s.configMutated[name] = true
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
	if unary, isUnary := expr.(*golang.Unary); isUnary && unary.Operator.Element == golang.AddressOf {
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
	return !s.configMutated[name] && s.configUses[name] == 2
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
	for name := range scan.configPathStyle {
		if scan.configRestructured[name] && declaresName(md, name) {
			return true
		}
	}
	return false
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
	if !isIdent || !v.scan.configPathStyle[recv.Name] {
		return nil, false
	}
	value, unwrapped := v.scan.pointerValueSource(a.Value.Element)
	if !unwrapped {
		return nil, false
	}
	return value, true
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
	if !v.scan.configPathStyle[name] {
		return out, true
	}
	out = append(out, &java.Assignment{
		Prefix:   java.Space{Whitespace: stmt.GetPrefix().Whitespace},
		Markers:  java.Markers{ID: uuid.New(), Entries: []java.Marker{golang.ShortVarDecl{Ident: uuid.New()}}},
		Variable: &java.Identifier{Name: pathStyleVar, Type: lstutil.NamedType("bool")},
		Value: java.LeftPadded[java.Expression]{
			Before:  java.SingleSpace,
			Element: &java.Literal{Prefix: java.SingleSpace, Value: false, Source: "false", Type: lstutil.NamedType("bool")},
		},
	})
	return out, true
}
