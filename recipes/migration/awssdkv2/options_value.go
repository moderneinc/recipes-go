/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package awssdkv2

import (
	"fmt"
	"strings"

	"github.com/moderneinc/recipes-go/recipes/internal/lstutil"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/parser"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// A session.Options built up as a value — held in a local, passed by pointer to
// functions that set its fields, as hypnoglow/helm-s3's functional options do —
// is v2's config.LoadOptions, which the loader takes as one option. The type
// follows wherever it is named, its literal becomes LoadOptions' fields, and
// each field write becomes the LoadOptions field holding the same setting.

const (
	loadOptionsType   = "LoadOptions"
	loadOptionsHelper = "awsCompatLoadOptions"
	assumeRoleHelper  = "awsCompatAssumeRoleOption"
	loadOptionsFile   = "zz_awssdkv2_loadoptions.go"
)

// optionsWrites map the settings a session.Options value is written through to
// the LoadOptions field holding them.
var optionsWrites = map[string]string{
	"Config.Region":           "Region",
	"Config.Endpoint":         "BaseEndpoint",
	"Config.Credentials":      "Credentials",
	"Config.HTTPClient":       "HTTPClient",
	"Profile":                 "SharedConfigProfile",
	"AssumeRoleTokenProvider": "TokenProvider",
	"AssumeRoleDuration":      "Duration",
}

// loadOptionFields map a loader option to the LoadOptions field it sets.
var loadOptionFields = map[string]string{
	"WithRegion":                    "Region",
	"WithBaseEndpoint":              "BaseEndpoint",
	"WithRetryMaxAttempts":          "RetryMaxAttempts",
	"WithHTTPClient":                "HTTPClient",
	withCredentialsProvider:         "Credentials",
	withSharedConfigProfile:         "SharedConfigProfile",
	withSharedCredentialsFiles:      "SharedCredentialsFiles",
	withAssumeRoleCredentialOptions: "AssumeRoleCredentialOptions",
	"WithRetryer":                   "Retryer",
}

// isOptionsPointer reports whether expr is the type *session.Options.
func (s *fileScan) isOptionsPointer(expr java.Expression) bool {
	ptr, isPointer := expr.(*golang.PointerType)
	if !isPointer {
		return false
	}
	fa, isField := ptr.Elem.(*java.FieldAccess)
	if !isField {
		return false
	}
	name, isSession := qualifiedRef(fa, s.sessionPkg)
	return isSession && name == sessionOptions
}

// writePath reads `x.Config.Region` as x and "Config.Region".
func writePath(expr java.Expression) (*java.Identifier, string, bool) {
	var parts []string
	for {
		switch e := expr.(type) {
		case *java.FieldAccess:
			if e.Name.Element == nil {
				return nil, "", false
			}
			parts = append([]string{e.Name.Element.Name}, parts...)
			expr = e.Target
		case *java.Identifier:
			return e, strings.Join(parts, "."), len(parts) > 0
		default:
			return nil, "", false
		}
	}
}

// optionsValueNames finds the names holding a session.Options value whose every
// use the rewrite carries over, and reports whether every one does.
func (s *fileScan) optionsValueNames(cu *golang.CompilationUnit) (map[string]bool, bool) {
	scan := visitor.Init(&optionsValueScan{scan: s, declared: map[string]bool{}, accepted: map[string]int{}})
	scan.Visit(cu, nil)
	if scan.failed {
		return nil, false
	}
	for name := range scan.declared {
		if scan.accepted[name] != s.configUses[name] {
			return nil, false
		}
	}
	return scan.declared, true
}

type optionsValueScan struct {
	visitor.GoVisitor
	scan     *fileScan
	declared map[string]bool
	accepted map[string]int
	failed   bool
}

func (v *optionsValueScan) VisitVariableDeclarations(vd *java.VariableDeclarations, p any) java.J {
	if v.scan.isOptionsPointer(vd.TypeExpr) {
		for _, rp := range vd.Variables {
			if d := rp.Element; d != nil && d.Name != nil && d.Name.Name != "" && d.Name.Name != "_" {
				v.declared[d.Name.Name] = true
				v.accepted[d.Name.Name]++
			}
		}
	}
	return v.GoVisitor.VisitVariableDeclarations(vd, p)
}

func (v *optionsValueScan) VisitAssignment(a *java.Assignment, p any) java.J {
	if target, isIdent := a.Variable.(*java.Identifier); isIdent && java.HasMarker[golang.ShortVarDecl](a.Markers) {
		if comp, isOptions := v.scan.sessionOptionsLiteral(a.Value.Element); isOptions {
			if local, isBuilder := builderCarried(comp); isBuilder && v.scan.builderVars[local.Name] {
				v.declared[target.Name] = true
				v.accepted[target.Name]++
			} else if _, carried, ok := v.scan.optionsOrCarriedConfig(comp); ok && carried == nil {
				v.declared[target.Name] = true
				v.accepted[target.Name]++
			} else {
				v.failed = true
			}
		}
	} else if root, path, ok := writePath(a.Variable); ok && v.declared[root.Name] {
		if _, mapped := optionsWrites[path]; mapped {
			v.accepted[root.Name]++
		}
	}
	return v.GoVisitor.VisitAssignment(a, p)
}

func (v *optionsValueScan) VisitMethodInvocation(mi *java.MethodInvocation, p any) java.J {
	for _, arg := range realArgs(mi) {
		if unary, isUnary := arg.(*golang.Unary); isUnary && unary.Operator.Element == golang.AddressOf {
			arg = unary.Expression
		}
		if id, isIdent := arg.(*java.Identifier); isIdent && v.declared[id.Name] {
			v.accepted[id.Name]++
		}
	}
	return v.GoVisitor.VisitMethodInvocation(mi, p)
}

// optionsValue reports whether expr names a session.Options value.
func (s *fileScan) optionsValue(expr java.Expression) (*java.Identifier, bool) {
	id, isIdent := expr.(*java.Identifier)
	return id, isIdent && s.optionsValues[id.Name]
}

// loadOptionsTypeRef names config.LoadOptions.
func (s *fileScan) loadOptionsTypeRef(prefix java.Space) *java.FieldAccess {
	return &java.FieldAccess{
		Prefix: prefix,
		Target: &java.Identifier{Name: s.configLocal, Type: lstutil.NamedType(v2Config)},
		Name:   java.LeftPadded[*java.Identifier]{Element: &java.Identifier{Name: loadOptionsType}},
		Type:   lstutil.NamedType(v2Config + "." + loadOptionsType),
	}
}

// loadOptionsLiteral builds the LoadOptions literal the loader options amount
// to.
func (s *fileScan) loadOptionsLiteral(options []configOption) (java.Expression, bool) {
	var fields []string
	values := map[string]java.Expression{}
	var serviceOptions []string
	for i, o := range options {
		placeholder := fmt.Sprintf("awsLoadOption%d", i)
		values[placeholder] = o.value
		if o.with == "WithServiceOptions" {
			serviceOptions = append(serviceOptions, placeholder)
			continue
		}
		field, ok := loadOptionFields[o.with]
		if !ok {
			return nil, false
		}
		fields = append(fields, fmt.Sprintf("\t%s: %s,\n", field, placeholder))
	}
	if len(serviceOptions) > 0 {
		fields = append(fields, fmt.Sprintf("\tServiceOptions: []func(string, any){%s},\n", strings.Join(serviceOptions, ", ")))
	}
	source := fmt.Sprintf("package p\n\nvar _ = %s.%s{\n%s}\n", s.configLocal, loadOptionsType, strings.Join(fields, ""))
	if len(fields) == 0 {
		source = fmt.Sprintf("package p\n\nvar _ = %s.%s{}\n", s.configLocal, loadOptionsType)
	}
	cu, err := parser.NewGoParser().Parse("options.go", source)
	if err != nil {
		return nil, false
	}
	lift := visitor.Init(&compositeLift{})
	lift.Visit(cu, nil)
	if lift.literal == nil {
		return nil, false
	}
	realign(lift.literal.Elements.Elements)
	var literal java.Expression = lift.literal
	for name, value := range values {
		bound, ok := visitor.Init(&placeholderBind{name: name, value: value}).Visit(literal, nil).(java.Expression)
		if !ok {
			return nil, false
		}
		literal = bound
	}
	return literal, true
}

type compositeLift struct {
	visitor.GoVisitor
	literal *golang.Composite
}

func (v *compositeLift) VisitComposite(comp *golang.Composite, p any) java.J {
	if v.literal == nil {
		v.literal = comp
	}
	return comp
}

// optionsWrite rewrites a write to a session.Options value's setting as the
// write to the LoadOptions field holding it.
func (v *migrateVisitor) optionsWrite(a *java.Assignment) (*java.Assignment, bool) {
	root, path, ok := writePath(a.Variable)
	if !ok || !v.scan.optionsValues[root.Name] {
		return nil, false
	}
	field, mapped := optionsWrites[path]
	if !mapped {
		return nil, false
	}
	value := a.Value.Element
	switch path {
	case "Config.Region", "Config.Endpoint":
		inner, unwrapped := v.scan.pointerValueSource(value)
		if !unwrapped {
			v.awsUsed, v.needsAws = true, true
			inner = &java.MethodInvocation{
				Select:     &java.RightPadded[java.Expression]{Element: &java.Identifier{Name: v.scan.awsQualifier(), Type: lstutil.NamedType(v2Aws)}},
				Name:       &java.Identifier{Name: "ToString"},
				Arguments:  java.Container[java.Expression]{Elements: []java.RightPadded[java.Expression]{{Element: lstutil.SetExprPrefix(value, java.EmptySpace)}}},
				MethodType: lstutil.FuncType(v2Aws, "ToString", nil),
			}
		}
		if path == "Config.Endpoint" {
			v.needsConfigHelper = true
			inner = endpointCall(inner)
		}
		value = inner
	case "AssumeRoleTokenProvider", "AssumeRoleDuration":
		// Added to whatever assume-role options the value already holds.
		var token, duration java.Expression
		if field == "TokenProvider" {
			token = value
		} else {
			duration = value
		}
		option, built := v.scan.assumeRoleOption(token, duration)
		if !built {
			return nil, false
		}
		if v.scan.stscredsPkg == "" {
			v.needsStscreds = true
		}
		v.needsLoadOptions = true
		current := &java.FieldAccess{
			Target: lstutil.SetExprPrefix(root, java.EmptySpace),
			Name:   java.LeftPadded[*java.Identifier]{Element: &java.Identifier{Name: "AssumeRoleCredentialOptions"}},
		}
		value = helperCallOf(assumeRoleHelper, current, option.value)
		field = "AssumeRoleCredentialOptions"
	}
	c := *a
	c.Variable = &java.FieldAccess{
		Prefix: a.Variable.GetPrefix(),
		Target: lstutil.SetExprPrefix(root, java.EmptySpace),
		Name:   java.LeftPadded[*java.Identifier]{Element: &java.Identifier{Name: field}},
	}
	c.Value = java.LeftPadded[java.Expression]{Before: a.Value.Before, Element: lstutil.SetExprPrefix(value, a.Value.Element.GetPrefix()), Markers: a.Value.Markers}
	return &c, true
}

const loadOptionsSource = `// Code generated by MigrateAwsSdkGoToV2. DO NOT EDIT.

package %s

import (
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
)

// awsCompatLoadOptions loads a config from options built up as a value, as
// aws-sdk-go v1's session.NewSessionWithOptions took a session.Options.
func awsCompatLoadOptions(options awsconfig.LoadOptions) func(*awsconfig.LoadOptions) error {
	return func(o *awsconfig.LoadOptions) error {
		*o = options
		return nil
	}
}

// awsCompatAssumeRoleOption adds a setting to the assume-role options a value
// already holds, which v1 kept as separate fields.
func awsCompatAssumeRoleOption(current, setting func(*stscreds.AssumeRoleOptions)) func(*stscreds.AssumeRoleOptions) {
	return func(o *stscreds.AssumeRoleOptions) {
		if current != nil {
			current(o)
		}
		setting(o)
	}
}
`

// optionsLiteralRefScan records the session references in the literals of
// session.Options locals, which their rewrite consumes.
type optionsLiteralRefScan struct {
	visitor.GoVisitor
	scan *fileScan
}

func (v *optionsLiteralRefScan) VisitAssignment(a *java.Assignment, p any) java.J {
	if target, isIdent := a.Variable.(*java.Identifier); isIdent && v.scan.optionsValues[target.Name] {
		if comp, isOptions := v.scan.sessionOptionsLiteral(a.Value.Element); isOptions {
			visitor.Init(&sessionRefScan{scan: v.scan}).Visit(comp, nil)
		}
	}
	return v.GoVisitor.VisitAssignment(a, p)
}

// optionsLocalLiteral rewrites the literal a session.Options local is declared
// from as the LoadOptions literal it amounts to.
func (v *migrateVisitor) optionsLocalLiteral(a *java.Assignment) (*java.Assignment, bool) {
	target, isIdent := a.Variable.(*java.Identifier)
	if !isIdent || !v.scan.optionsValues[target.Name] || !java.HasMarker[golang.ShortVarDecl](a.Markers) {
		return nil, false
	}
	comp, isOptions := v.scan.sessionOptionsLiteral(a.Value.Element)
	if !isOptions {
		return nil, false
	}
	// A config built through setters answers the options value itself.
	if local, isBuilder := builderCarried(comp); isBuilder && v.scan.builderVars[local.Name] {
		v.needsConfigBuilder, v.needsDisableSSL, v.needsPathStyle = true, true, true
		c := *a
		c.Value = java.LeftPadded[java.Expression]{Before: a.Value.Before, Element: &java.MethodInvocation{
			Prefix:     a.Value.Element.GetPrefix(),
			Select:     &java.RightPadded[java.Expression]{Element: &java.Identifier{Name: local.Name, Type: lstutil.NamedType(builderType)}},
			Name:       &java.Identifier{Name: "options"},
			MethodType: lstutil.FuncType(builderType, "options", nil),
		}, Markers: a.Value.Markers}
		return &c, true
	}
	options, carried, ok := v.scan.optionsOrCarriedConfig(comp)
	if !ok || carried != nil {
		return nil, false
	}
	literal, built := v.scan.loadOptionsLiteral(options)
	if !built {
		return nil, false
	}
	v.flagOptions(options)
	c := *a
	c.Value = java.LeftPadded[java.Expression]{Before: a.Value.Before, Element: lstutil.SetExprPrefix(literal, a.Value.Element.GetPrefix()), Markers: a.Value.Markers}
	return &c, true
}

// optionsLoad loads the config a session.Options local holds.
func (v *migrateVisitor) optionsLoad(mi *java.MethodInvocation, ctx java.Expression, local *java.Identifier) java.Expression {
	v.needsLoadOptions = true
	return &java.MethodInvocation{
		Prefix:     mi.Prefix,
		Select:     &java.RightPadded[java.Expression]{Element: &java.Identifier{Name: v.scan.configLocal, Type: lstutil.NamedType(v2Config)}},
		Name:       &java.Identifier{Name: "LoadDefaultConfig"},
		MethodType: lstutil.FuncType(v2Config, "LoadDefaultConfig", nil),
		Arguments: java.Container[java.Expression]{Elements: []java.RightPadded[java.Expression]{
			{Element: lstutil.SetExprPrefix(ctx, java.EmptySpace)},
			{Element: helperCallOf(loadOptionsHelper, lstutil.SetExprPrefix(local, java.SingleSpace))},
		}},
	}
}

// optionsAssignment reports whether a declares a session.Options local or
// writes one of its settings.
func (s *fileScan) optionsAssignment(a *java.Assignment) bool {
	if !s.optionsValuesOK {
		return false
	}
	if target, isIdent := a.Variable.(*java.Identifier); isIdent {
		return s.optionsValues[target.Name] && java.HasMarker[golang.ShortVarDecl](a.Markers)
	}
	root, path, ok := writePath(a.Variable)
	_, mapped := optionsWrites[path]
	return ok && mapped && s.optionsValues[root.Name]
}
