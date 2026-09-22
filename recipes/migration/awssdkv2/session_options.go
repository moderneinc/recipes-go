/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package awssdkv2

import (
	"github.com/moderneinc/recipes-go/recipes/internal/lstutil"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// session.Options configured the session v2 configures through LoadDefaultConfig
// options. Most of what it carried is gone — v2 always reads the shared config,
// and the handler stack and endpoint resolver are configured elsewhere — so only
// the fields with a loader option carry over.
const (
	assumeRoleTokenProvider         = "AssumeRoleTokenProvider"
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
			option, built := s.tokenProviderOption(kv.Value.Element)
			if !built {
				return nil, nil, false
			}
			options = append(options, option)
		case sessionConfig:
			if nested, isLiteral := s.configLiteralOptions(kv.Value.Element); isLiteral {
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
	if carried != nil && len(options) > 0 {
		// A carried config and a loader option cannot both apply: the loader
		// would build a config of its own and throw the carried one away.
		return nil, nil, false
	}
	return options, carried, true
}

// tokenProviderOption wraps v1's MFA token provider in the assume-role options
// v2 configures it through.
func (s *fileScan) tokenProviderOption(value java.Expression) (configOption, bool) {
	if s.stscredsPkg == "" {
		return configOption{}, false
	}
	literal, built := tokenProviderLiteral(s.stscredsPkg, value)
	if !built {
		return configOption{}, false
	}
	return configOption{with: withAssumeRoleCredentialOptions, value: literal}, true
}

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

// loadConfigFromOptions rewrites session.NewSessionWithOptions as the v2 config
// load the options it was given amount to.
func (v *migrateVisitor) loadConfigFromOptions(mi *java.MethodInvocation) java.J {
	args := realArgs(mi)
	if len(args) != 1 {
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
		v.needsConfigHelper = true
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
