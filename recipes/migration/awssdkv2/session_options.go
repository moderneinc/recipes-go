/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package awssdkv2

import (
	"fmt"
	"github.com/moderneinc/recipes-go/recipes/internal/lstutil"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/parser"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
	"strings"
)

// session.Options configured the session v2 configures through LoadDefaultConfig
// options. Most of what it carried is gone — v2 always reads the shared config,
// and the handler stack and endpoint resolver are configured elsewhere — so only
// the fields with a loader option carry over.
const (
	assumeRoleTokenProvider         = "AssumeRoleTokenProvider"
	assumeRoleDuration              = "AssumeRoleDuration"
	withAssumeRoleCredentialOptions = "WithAssumeRoleCredentialOptions"

	sessionOptions    = "Options"
	sharedConfigState = "SharedConfigState"
	sharedProfile     = "Profile"
	sessionConfig     = "Config"
)

// sessionOptionNames are the session names a converted Options literal spells,
// which the guard allows through on that account.
var sessionOptionNames = map[string]bool{
	sessionOptions:             true,
	"SharedConfigEnable":       true,
	"SharedConfigDisable":      true,
	"SharedConfigStateFromEnv": true,
}

// sessionOptionsLiteral reports whether expr is a `session.Options{…}` literal.
func (s *fileScan) sessionOptionsLiteral(expr java.Expression) (*golang.Composite, bool) {
	comp, isComposite := expr.(*golang.Composite)
	if !isComposite {
		return nil, false
	}
	fa, isField := comp.TypeExpr.(*java.FieldAccess)
	if !isField {
		return nil, false
	}
	name, isSession := qualifiedRef(fa, s.sessionPkg)
	if !isSession || name != sessionOptions {
		return nil, false
	}
	return comp, true
}

// convertibleOptions reports whether a session.Options literal carries only what
// a v2 config load can be given, returning the profile and region it sets.
func (s *fileScan) convertibleOptions(comp *golang.Composite) ([]configOption, bool) {
	options, _, ok := s.optionsOrCarriedConfig(comp)
	return options, ok
}

// optionsOrCarriedConfig reads a session.Options literal. It either amounts to
// a list of loader options, or names a config the caller already holds — which
// v2 takes as it stands, and which nothing else can be mixed with.
func (s *fileScan) optionsOrCarriedConfig(comp *golang.Composite) (options []configOption, carried java.Expression, ok bool) {
	var tokenProvider, duration java.Expression
	for _, rp := range comp.Elements.Elements {
		if _, isEmpty := rp.Element.(*java.Empty); isEmpty {
			continue
		}
		kv, isKeyValue := rp.Element.(*golang.KeyValue)
		if !isKeyValue {
			return nil, nil, false
		}
		key, isIdent := kv.Key.(*java.Identifier)
		if !isIdent {
			return nil, nil, false
		}
		switch key.Name {
		case sharedConfigState:
			// v2 always reads the shared config, so enabling it is the default
			// and disabling it has no option to ask for.
			if !s.enablesSharedConfig(kv.Value.Element) {
				return nil, nil, false
			}
		case sharedProfile:
			options = append(options, configOption{with: withSharedConfigProfile, value: kv.Value.Element})
		case assumeRoleTokenProvider:
			tokenProvider = kv.Value.Element
		case assumeRoleDuration:
			duration = kv.Value.Element
		case sessionConfig:
			if nested, isLiteral := s.sessionArgOptions(kv.Value.Element); isLiteral {
				options = append(options, nested...)
				continue
			}
			// A config the file already holds is the v2 config; it is carried
			// whole rather than taken apart into loader options.
			carried = kv.Value.Element
		default:
			return nil, nil, false
		}
	}
	// v2 holds one function setting the assume-role options, so both settings
	// go in it.
	if tokenProvider != nil || duration != nil {
		option, built := s.assumeRoleOption(tokenProvider, duration)
		if !built {
			return nil, nil, false
		}
		options = append(options, option)
	}
	if carried != nil && len(options) > 0 {
		// A carried config and a loader option cannot both apply: the loader
		// would build a config of its own and throw the carried one away.
		return nil, nil, false
	}
	return options, carried, true
}

// assumeRoleOption sets v2's assume-role options to v1's MFA token provider and
// role duration, whichever were given.
func (s *fileScan) assumeRoleOption(tokenProvider, duration java.Expression) (configOption, bool) {
	local := s.stscredsPkg
	if local == "" {
		// The v2 package is imported where the option is written.
		if s.bindsStscreds {
			return configOption{}, false
		}
		local = "stscreds"
	}
	var body []string
	if tokenProvider != nil {
		body = append(body, "o.TokenProvider = "+tokenProviderPlaceholder)
	}
	if duration != nil {
		body = append(body, "o.Duration = "+durationPlaceholder)
	}
	cu, err := parser.NewGoParser().Parse("assume.go", fmt.Sprintf("package p\n\nvar _ = func(o *%s.AssumeRoleOptions) {\n\t%s\n}\n", local, strings.Join(body, "\n\t")))
	if err != nil {
		return configOption{}, false
	}
	lift := visitor.Init(&funcLiteralLift{})
	lift.Visit(cu, nil)
	if lift.literal == nil {
		return configOption{}, false
	}
	literal := lift.literal
	for name, value := range map[string]java.Expression{tokenProviderPlaceholder: tokenProvider, durationPlaceholder: duration} {
		if value == nil {
			continue
		}
		bound, ok := visitor.Init(&placeholderBind{name: name, value: value}).Visit(literal, nil).(java.Expression)
		if !ok {
			return configOption{}, false
		}
		literal = bound
	}
	return configOption{with: withAssumeRoleCredentialOptions, value: literal}, true
}

const durationPlaceholder = "awsDurationPlaceholder"

func (s *fileScan) enablesSharedConfig(expr java.Expression) bool {
	fa, isField := expr.(*java.FieldAccess)
	if !isField {
		return false
	}
	name, isSession := qualifiedRef(fa, s.sessionPkg)
	return isSession && name == "SharedConfigEnable"
}

// sessionOptionsScan asks of every session.Options literal in the file whether
// it converts, so a reference to the type can be allowed on the strength of the
// literals it belongs to rather than blocked on sight.
type sessionOptionsScan struct {
	visitor.GoVisitor
	scan *fileScan
}

func (v *sessionOptionsScan) VisitComposite(comp *golang.Composite, p any) java.J {
	if _, isOptions := v.scan.sessionOptionsLiteral(comp); isOptions {
		if _, ok := v.scan.convertibleOptions(comp); !ok {
			v.scan.sessionOptionsBlocked = true
		}
	}
	return v.GoVisitor.VisitComposite(comp, p)
}

func (v *sessionOptionsScan) VisitMethodInvocation(mi *java.MethodInvocation, p any) java.J {
	if call, ok := qualifiedCall(mi, v.scan.sessionPkg); ok && call == "NewSessionWithOptions" {
		if args := realArgs(mi); len(args) == 1 {
			if comp, isOptions := v.scan.sessionOptionsLiteral(args[0]); isOptions {
				refs := visitor.Init(&sessionRefScan{scan: v.scan})
				refs.Visit(comp, nil)
			}
		}
	}
	return v.GoVisitor.VisitMethodInvocation(mi, p)
}

// sessionRefScan records the session references within a literal the rewrite
// consumes.
type sessionRefScan struct {
	visitor.GoVisitor
	scan *fileScan
}

func (v *sessionRefScan) VisitFieldAccess(fa *java.FieldAccess, p any) java.J {
	if _, ok := qualifiedRef(fa, v.scan.sessionPkg); ok {
		if v.scan.consumedOptionRefs == nil {
			v.scan.consumedOptionRefs = map[*java.FieldAccess]bool{}
		}
		v.scan.consumedOptionRefs[fa] = true
	}
	return v.GoVisitor.VisitFieldAccess(fa, p)
}

// loadConfigFromOptions rewrites session.NewSessionWithOptions as the v2 config
// load the options it was given amount to.
func (v *migrateVisitor) loadConfigFromOptions(mi *java.MethodInvocation) java.J {
	args := realArgs(mi)
	if len(args) != 1 {
		return mi
	}
	if local, isValue := v.scan.optionsValue(args[0]); isValue {
		if ctx := v.contextExpr(); ctx != nil {
			return v.optionsLoad(mi, ctx, local)
		}
		return mi
	}
	comp, isOptions := v.scan.sessionOptionsLiteral(args[0])
	if !isOptions {
		return mi
	}
	options, carried, ok := v.scan.optionsOrCarriedConfig(comp)
	if !ok {
		return mi
	}
	// A config the caller already holds is what v2 would have built, so the
	// session call becomes the pass-through that keeps the error the v1 one
	// answered with.
	if carried != nil {
		v.carriedConfigs++
		c := *mi
		c.Select = nil
		c.Name = &java.Identifier{Name: sessionConfigHelper}
		c.MethodType = lstutil.FuncType("", sessionConfigHelper, nil)
		c.Arguments = mi.Arguments
		c.Arguments.Elements = []java.RightPadded[java.Expression]{
			{Element: lstutil.SetExprPrefix(carried, java.EmptySpace)},
		}
		return &c
	}
	ctx := v.contextExpr()
	if ctx == nil {
		return mi
	}
	if applied, built := v.loadConfigCall(ctx, options); built {
		return applied
	}
	return mi
}

// sessionVarScan records the names declared as a *session.Session. v2 replaces
// the pointer with a config value, so a nil test on one has nothing left to
// compare against.
type sessionVarScan struct {
	visitor.GoVisitor
	scan *fileScan
}

func (v *sessionVarScan) VisitVariableDeclarations(vd *java.VariableDeclarations, p any) java.J {
	if v.scan.declaresSession(vd.TypeExpr) {
		for _, decl := range vd.Variables {
			if d := decl.Element; d != nil && d.Name != nil {
				v.scan.sessionVars[d.Name.Name] = true
			}
		}
	}
	return v.GoVisitor.VisitVariableDeclarations(vd, p)
}

// A session is as often assigned straight from its constructor, whose result
// the migration turns into the config the variable then holds.
func (v *sessionVarScan) VisitAssignment(a *java.Assignment, p any) java.J {
	a = v.GoVisitor.VisitAssignment(a, p).(*java.Assignment)
	if name, isIdent := a.Variable.(*java.Identifier); isIdent && v.scan.constructsSession(a.Value.Element) {
		v.scan.sessionVars[name.Name] = true
	}
	return a
}

func (v *sessionVarScan) VisitMultiAssignment(ma *golang.MultiAssignment, p any) java.J {
	ma = v.GoVisitor.VisitMultiAssignment(ma, p).(*golang.MultiAssignment)
	if len(ma.Variables) == 2 && len(ma.Values) == 1 && v.scan.constructsSession(ma.Values[0].Element) {
		if name, isIdent := ma.Variables[0].Element.(*java.Identifier); isIdent {
			v.scan.sessionVars[name.Name] = true
		}
	}
	return ma
}

// sessionConstructors are the session package's functions returning one.
var sessionConstructors = map[string]bool{"Must": true, "New": true, "NewSession": true, "NewSessionWithOptions": true}

// constructsSession reports whether expr is a call building a session.
func (s *fileScan) constructsSession(expr java.Expression) bool {
	mi, isCall := expr.(*java.MethodInvocation)
	if !isCall {
		return false
	}
	call, ok := qualifiedCall(mi, s.sessionPkg)
	return ok && sessionConstructors[call]
}

func (v *sessionVarScan) VisitBinary(bin *java.Binary, p any) java.J {
	bin = v.GoVisitor.VisitBinary(bin, p).(*java.Binary)
	switch bin.Operator.Element {
	case java.Equal, java.NotEqual:
	default:
		return bin
	}
	if v.scan.comparesSessionToNil(bin.Left, bin.Right) || v.scan.comparesSessionToNil(bin.Right, bin.Left) {
		v.scan.sessionNilCompared = true
	}
	return bin
}

func (s *fileScan) comparesSessionToNil(name, nilSide java.Expression) bool {
	id, isIdent := name.(*java.Identifier)
	if !isIdent || !s.sessionVars[id.Name] {
		return false
	}
	other, isIdent := nilSide.(*java.Identifier)
	return isIdent && other.Name == "nil"
}

// declaresSession reports whether a type expression is v1's *session.Session.
func (s *fileScan) declaresSession(expr java.Expression) bool {
	if s.sessionPkg == "" {
		return false
	}
	if ptr, isPointer := expr.(*golang.PointerType); isPointer {
		expr = ptr.Elem
	}
	fa, isField := expr.(*java.FieldAccess)
	if !isField {
		return false
	}
	name, isSession := qualifiedRef(fa, s.sessionPkg)
	return isSession && name == "Session"
}

// A function returning a session returned nil beside its error. v2's config is
// a value, so the zero config stands where the nil did.

// sessionResults returns the positions of md's results declared as a session.
func (s *fileScan) sessionResults(md *java.MethodDeclaration) map[int]bool {
	positions := map[int]bool{}
	if md.ReturnType == nil || s.sessionPkg == "" {
		return positions
	}
	list, isList := md.ReturnType.(*golang.TypeList)
	if !isList {
		if s.declaresSession(md.ReturnType) {
			positions[0] = true
		}
		return positions
	}
	i := 0
	for _, rp := range list.Types.Elements {
		switch e := rp.Element.(type) {
		case *java.VariableDeclarations:
			for range e.Variables {
				if s.declaresSession(e.TypeExpr) {
					positions[i] = true
				}
				i++
			}
		default:
			if expr, isExpr := rp.Element.(java.Expression); isExpr && s.declaresSession(expr) {
				positions[i] = true
			}
			i++
		}
	}
	return positions
}

// zeroConfig builds `aws.Config{}`.
func (v *migrateVisitor) zeroConfig(prefix java.Space) java.Expression {
	v.awsUsed, v.needsAws = true, true
	return &golang.Composite{
		Prefix: prefix,
		TypeExpr: &java.FieldAccess{
			Target: &java.Identifier{Name: v.scan.awsQualifier(), Type: lstutil.NamedType(v2Aws)},
			Name:   java.LeftPadded[*java.Identifier]{Element: &java.Identifier{Name: "Config"}},
			Type:   lstutil.NamedType(v2Aws + ".Config"),
		},
	}
}

func isNil(expr java.Expression) bool {
	id, isIdent := expr.(*java.Identifier)
	return isIdent && id.Name == "nil"
}

func (v *migrateVisitor) VisitGoReturn(r *golang.Return, p any) java.J {
	r = v.GoVisitor.VisitGoReturn(r, p).(*golang.Return)
	if len(v.sessionResults) == 0 {
		return r
	}
	var out []java.RightPadded[java.Expression]
	for i, rp := range r.Expressions {
		if v.sessionResults[i] && isNil(rp.Element) {
			if out == nil {
				out = append([]java.RightPadded[java.Expression](nil), r.Expressions...)
			}
			out[i].Element = v.zeroConfig(rp.Element.GetPrefix())
		}
	}
	if out == nil {
		return r
	}
	c := *r
	c.Expressions = out
	return &c
}

func (v *migrateVisitor) VisitReturn(r *java.Return, p any) java.J {
	r = v.GoVisitor.VisitReturn(r, p).(*java.Return)
	if v.sessionResults[0] && r.Expression != nil && isNil(r.Expression) {
		c := *r
		c.Expression = v.zeroConfig(r.Expression.GetPrefix())
		return &c
	}
	return r
}
