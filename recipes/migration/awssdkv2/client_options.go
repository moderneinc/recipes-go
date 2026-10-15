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

func (v *funcLiteralLift) VisitMethodDeclaration(md *java.MethodDeclaration, p any) java.J {
	if v.literal == nil && (md.Name == nil || md.Name.Name == "") {
		v.literal = md
	}
	return v.GoVisitor.VisitMethodDeclaration(md, p)
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

// telia-oss/aws-env's shape: a client takes a trailing config held in a local
// that is either nil or a literal setting the region. v1 skipped a nil config
// and took the region from any other, which the option does too.
const optionalRegionSource = `package p

var _ = func(o *%s.Options) {
	if %s != nil {
		o.Region = %s.Region
	}
}
`

// optionalRegionOption builds the option overriding the region from the
// config held in name, when there is one.
func optionalRegionOption(serviceLocal, name string) (java.Expression, bool) {
	cu, err := parser.NewGoParser().Parse("opt.go", fmt.Sprintf(optionalRegionSource, serviceLocal, name, name))
	if err != nil {
		return nil, false
	}
	lift := visitor.Init(&funcLiteralLift{})
	lift.Visit(cu, nil)
	return lift.literal, lift.literal != nil
}

// regionConfigScan finds the locals holding nothing but a nil or a
// region-only aws.Config literal, read only as a client constructor's
// trailing config. A name used any other way is disqualified.
type regionConfigScan struct {
	visitor.GoVisitor
	scan  *fileScan
	uses  map[string]int
	clean map[string]int
	bad   map[string]bool
}

func (v *regionConfigScan) VisitVariableDeclarations(vd *java.VariableDeclarations, p any) java.J {
	ptr, isPointer := vd.TypeExpr.(*golang.PointerType)
	if isPointer {
		if fa, isField := ptr.Elem.(*java.FieldAccess); isField {
			if name, isAws := qualifiedRef(fa, v.scan.awsPkg); isAws && name == "Config" {
				for _, rp := range vd.Variables {
					if d := rp.Element; d != nil && d.Name != nil {
						v.clean[d.Name.Name]++
						if d.Initializer != nil && !v.regionOnly(d.Initializer.Element) {
							v.bad[d.Name.Name] = true
						}
					}
				}
			}
		}
	}
	return v.GoVisitor.VisitVariableDeclarations(vd, p)
}

func (v *regionConfigScan) VisitAssignment(a *java.Assignment, p any) java.J {
	if id, isIdent := a.Variable.(*java.Identifier); isIdent && v.clean[id.Name] > 0 {
		v.clean[id.Name]++
		if !v.regionOnly(a.Value.Element) {
			v.bad[id.Name] = true
		}
	}
	return v.GoVisitor.VisitAssignment(a, p)
}

func (v *regionConfigScan) VisitMethodInvocation(mi *java.MethodInvocation, p any) java.J {
	if _, isConstructor := v.scan.clientConstructorService(mi); isConstructor {
		if args := realArgs(mi); len(args) == 2 {
			if id, isIdent := args[1].(*java.Identifier); isIdent {
				v.clean[id.Name]++
			}
		}
	}
	return v.GoVisitor.VisitMethodInvocation(mi, p)
}

func (v *regionConfigScan) VisitIdentifier(id *java.Identifier, p any) java.J {
	v.uses[id.Name]++
	return id
}

// regionOnly reports whether expr is nil or `&aws.Config{Region: …}`.
func (v *regionConfigScan) regionOnly(expr java.Expression) bool {
	if id, isIdent := expr.(*java.Identifier); isIdent {
		return id.Name == "nil"
	}
	options, ok := v.scan.configLiteralOptions(expr)
	return ok && len(options) == 1 && options[0].with == "WithRegion"
}

// regionConfigLocals runs the scan over cu.
func (s *fileScan) regionConfigLocals(cu *golang.CompilationUnit) map[string]bool {
	scan := visitor.Init(&regionConfigScan{scan: s, uses: map[string]int{}, clean: map[string]int{}, bad: map[string]bool{}})
	scan.Visit(cu, nil)
	locals := map[string]bool{}
	for name, clean := range scan.clean {
		// Every appearance of the name is the declaration, an assignment or a
		// constructor argument.
		if !scan.bad[name] && clean > 1 && clean == scan.uses[name] {
			locals[name] = true
		}
	}
	return locals
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
	if len(args) != 2 {
		return false
	}
	if id, isIdent := args[1].(*java.Identifier); isIdent && s.regionConfigs[id.Name] {
		return true
	}
	_, ok := s.clientConfigRegion(args[1])
	return ok
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
