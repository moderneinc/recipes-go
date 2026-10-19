/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package awssdkv2

import (
	"fmt"
	"strings"
	"sync"

	"github.com/moderneinc/recipes-go/recipes/internal/lstutil"
	"github.com/moderneinc/recipes-go/recipes/migration/awssdkv2/awsexportdata"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/template"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// v1 configured a session by filling in an aws.Config; v2 passes functional
// options to the loader. The fields with a loader option carry over one for one,
// and the rest — the endpoint resolver, the handler stack, anything a client
// rather than the config owns — leave the file alone.
type configOption struct {
	// with is the loader option's name, `WithRegion` and its siblings.
	with string
	// value is the argument it takes, already unwrapped from the pointer v1
	// needed.
	value java.Expression
	// usesTime records that the value names the time package.
	usesTime bool
}

// configFieldOptions map an aws.Config field to the loader option that replaces
// it. A field absent from this table has no counterpart the recipe writes.
var configFieldOptions = map[string]string{
	"Region":     "WithRegion",
	"Endpoint":   "WithBaseEndpoint",
	"MaxRetries": "WithRetryMaxAttempts",
	"HTTPClient": "WithHTTPClient",
	// Applied to S3's clients through the service options the config carries.
	s3ForcePathStyle: "WithServiceOptions",
}

const (
	credentialsField           = "Credentials"
	withCredentialsProvider    = "WithCredentialsProvider"
	withSharedCredentialsFiles = "WithSharedCredentialsFiles"
	withSharedConfigProfile    = "WithSharedConfigProfile"
)

var awsOpts = []*template.Capture{
	template.Expr("awsOpt0"), template.Expr("awsOpt1"), template.Expr("awsOpt2"),
	template.Expr("awsOpt3"), template.Expr("awsOpt4"), template.Expr("awsOpt5"),
}

var (
	loadConfigMu    sync.Mutex
	loadConfigCache = map[string]*template.GoTemplate{}
)

// loadConfigTemplateFor builds `config.LoadDefaultConfig(ctx, config.WithX(…), …)`
// for one sequence of options. The option names are part of the source, so there
// is one template per sequence.
func loadConfigTemplateFor(withs []string) (*template.GoTemplate, bool) {
	if len(withs) > len(awsOpts) {
		return nil, false
	}
	key := strings.Join(withs, ",")

	loadConfigMu.Lock()
	defer loadConfigMu.Unlock()
	if t, ok := loadConfigCache[key]; ok {
		return t, t != nil
	}
	loadConfigCache[key] = nil

	captures := []*template.Capture{awsCtx}
	parts := []string{awsCtx.String()}
	for i, with := range withs {
		captures = append(captures, awsOpts[i])
		parts = append(parts, fmt.Sprintf("config.%s(%s)", with, awsOpts[i]))
	}
	t := template.ExpressionTemplate(fmt.Sprintf("config.LoadDefaultConfig(%s)", strings.Join(parts, ", "))).
		Captures(captures...).
		Imports(v2Config).
		ExportData(awsexportdata.FS).
		Build()
	if t == nil {
		return nil, false
	}
	loadConfigCache[key] = t
	return t, true
}

// endpointCall wraps an endpoint in the generated helper giving it v1's scheme.
func endpointCall(endpoint java.Expression) java.Expression {
	return &java.MethodInvocation{
		Prefix:     endpoint.GetPrefix(),
		Name:       &java.Identifier{Name: endpointHelper},
		Arguments:  java.Container[java.Expression]{Elements: []java.RightPadded[java.Expression]{{Element: lstutil.SetExprPrefix(endpoint, java.EmptySpace)}}},
		MethodType: lstutil.FuncType("", endpointHelper, nil),
	}
}

// configChainOptions reads `aws.NewConfig().WithX(…)…`, returning the loader
// options it amounts to. Its setters take the value itself where the literal's
// fields took a pointer, and asking for STS's regional endpoints needs nothing:
// v2's are always regional.
func (s *fileScan) configChainOptions(expr java.Expression) ([]configOption, bool) {
	var options []configOption
	var disableSSL java.Expression
	for {
		mi, isCall := expr.(*java.MethodInvocation)
		if !isCall || mi.Name == nil {
			return nil, false
		}
		// request.WithRetryer(cfg, r) sets the retryer on the config it is
		// handed, as a setter would.
		if call, isRequest := qualifiedCall(mi, s.requestPkg); isRequest && call == "WithRetryer" && len(realArgs(mi)) == 2 {
			retryer, ok := s.configFieldOption(retryerField, realArgs(mi)[1])
			if !ok {
				return nil, false
			}
			options = append(options, retryer...)
			expr = realArgs(mi)[0]
			continue
		}
		if helper, isAws := qualifiedCall(mi, s.awsPkg); isAws {
			if helper != "NewConfig" || len(realArgs(mi)) != 0 {
				return nil, false
			}
			// The chain was read from its last setter back to its head.
			for i, j := 0, len(options)-1; i < j; i, j = i+1, j-1 {
				options[i], options[j] = options[j], options[i]
			}
			if disableSSL != nil {
				return withDisableSSL(options, disableSSL)
			}
			return options, true
		}
		args := realArgs(mi)
		if mi.Select == nil || len(args) != 1 {
			return nil, false
		}
		switch mi.Name.Name {
		case "WithRegion":
			options = append(options, configOption{with: "WithRegion", value: args[0]})
		case "WithEndpoint":
			options = append(options, configOption{with: "WithBaseEndpoint", value: endpointCall(args[0])})
		case "WithMaxRetries":
			options = append(options, configOption{with: "WithRetryMaxAttempts", value: shifted(args[0], configTargets["MaxRetries"].offset)})
		case "WithCredentials":
			options = append(options, configOption{with: withCredentialsProvider, value: args[0]})
		case "WithS3ForcePathStyle":
			options = append(options, configOption{with: "WithServiceOptions", value: pathStyleCall(args[0])})
		case "WithDisableSSL":
			// A later setter overrides an earlier one, and the chain is read
			// from the last.
			if disableSSL == nil {
				disableSSL = args[0]
			}
		case "WithCredentialsChainVerboseErrors":
			// v2 resolves one provider and reports its failure in full, so
			// asking for the chain's verbose errors needs nothing.
		case "WithSTSRegionalEndpoint":
			if !s.regionalSTSEndpoint(args[0]) {
				return nil, false
			}
		default:
			return nil, false
		}
		expr = mi.Select.Element
	}
}

// regionalSTSEndpoint reports whether expr is endpoints.RegionalSTSEndpoint.
func (s *fileScan) regionalSTSEndpoint(expr java.Expression) bool {
	fa, isField := expr.(*java.FieldAccess)
	if !isField {
		return false
	}
	name, isEndpoints := qualifiedRef(fa, s.endpointsPkg)
	return isEndpoints && name == "RegionalSTSEndpoint"
}

// configLiteralOptions reads an aws.Config literal, or a NewConfig chain,
// returning the loader options it amounts to. Either may be written with or
// without its address taken or dereferenced.
func (s *fileScan) configLiteralOptions(expr java.Expression) ([]configOption, bool) {
	if unary, isUnary := expr.(*golang.Unary); isUnary {
		expr = unary.Expression
	}
	if options, ok := s.configChainOptions(expr); ok {
		return options, true
	}
	comp, isComposite := expr.(*golang.Composite)
	if !isComposite {
		return nil, false
	}
	fa, isField := comp.TypeExpr.(*java.FieldAccess)
	if !isField {
		return nil, false
	}
	if name, isAws := qualifiedRef(fa, s.awsPkg); !isAws || name != "Config" {
		return nil, false
	}

	var options []configOption
	var disableSSL java.Expression
	for _, rp := range comp.Elements.Elements {
		if _, isEmpty := rp.Element.(*java.Empty); isEmpty {
			continue
		}
		kv, isKeyValue := rp.Element.(*golang.KeyValue)
		if !isKeyValue {
			return nil, false
		}
		key, isIdent := kv.Key.(*java.Identifier)
		if !isIdent {
			return nil, false
		}
		if key.Name == "CredentialsChainVerboseErrors" {
			continue
		}
		if key.Name == disableSSLField {
			inner, unwrapped := s.pointerValueSource(kv.Value.Element)
			if !unwrapped {
				return nil, false
			}
			disableSSL = inner
			continue
		}
		converted, ok := s.configFieldOption(key.Name, kv.Value.Element)
		if !ok {
			return nil, false
		}
		options = append(options, converted...)
	}
	if disableSSL != nil {
		return withDisableSSL(options, disableSSL)
	}
	return options, true
}

// configFieldOption maps one aws.Config field to the loader options it becomes.
// Credentials is the one that can yield two: a shared-credentials provider names
// both the file it reads and the profile within it.
func (s *fileScan) configFieldOption(field string, value java.Expression) ([]configOption, bool) {
	if field == credentialsField {
		return s.credentialsOptions(value)
	}
	if field == retryerField {
		fields, typeName, ok := s.retryerLiteral(value)
		if !ok {
			return nil, false
		}
		source := strings.ReplaceAll(s.retryerSource(fields, typeName), "aws.", s.awsQualifier()+".")
		parsed, ok := parsedExpression(source)
		if !ok {
			return nil, false
		}
		paths := map[string]string{"retry": v2Retry, "ratelimit": v2RateLimit, "time": "time", s.awsQualifier(): v2Aws}
		parsed = visitor.Init(&qualifiedCallTyping{paths: paths}).Visit(parsed, nil).(java.Expression)
		return []configOption{{with: "WithRetryer", value: parsed, usesTime: s.usesTime(fields)}}, true
	}
	with, mapped := configFieldOptions[field]
	if !mapped {
		return nil, false
	}
	if with == "WithHTTPClient" {
		// Already a value rather than a pointer in v1.
		return []configOption{{with: with, value: value}}, true
	}
	inner, unwrapped := s.pointerValueSource(value)
	if !unwrapped {
		return nil, false
	}
	if field == "Endpoint" {
		inner = endpointCall(inner)
	}
	if field == s3ForcePathStyle {
		inner = pathStyleCall(inner)
	}
	return []configOption{{with: with, value: shifted(inner, configTargets[field].offset)}}, true
}

// credentialsOptions maps v1's credential constructors to the loader options
// that stand in for them.
func (s *fileScan) credentialsOptions(value java.Expression) ([]configOption, bool) {
	// A provider held in a local is rewritten where it was built, or the file
	// is held back there.
	if _, isIdent := value.(*java.Identifier); isIdent {
		return []configOption{{with: withCredentialsProvider, value: value}}, true
	}
	mi, isCall := value.(*java.MethodInvocation)
	if !isCall {
		return nil, false
	}
	call, isCredentials := qualifiedCall(mi, s.credentialsPkg)
	if !isCredentials {
		return nil, false
	}
	args := realArgs(mi)
	switch call {
	case "NewStaticCredentials", "NewStaticCredentialsProvider":
		// The constructor itself is renamed on the way in, so what reaches here
		// is already a v2 provider.
		return []configOption{{with: withCredentialsProvider, value: value}}, true
	case "NewSharedCredentials":
		// v2 has no shared-credentials provider; the loader reads the file and
		// the profile itself.
		if len(args) != 2 {
			return nil, false
		}
		return []configOption{
			{with: withSharedCredentialsFiles, value: stringSlice(args[0])},
			{with: withSharedConfigProfile, value: args[1]},
		}, true
	}
	return nil, false
}

// stringSlice wraps an expression in `[]string{…}`, which is what the loader's
// shared-credentials option takes where v1 named one file.
func stringSlice(expr java.Expression) java.Expression {
	return &golang.Composite{
		TypeExpr: &java.ArrayType{ElementType: &java.Identifier{Name: "string", Type: lstutil.NamedType("string")}},
		Elements: java.Container[java.Expression]{
			Elements: []java.RightPadded[java.Expression]{{Element: lstutil.SetExprPrefix(expr, java.EmptySpace)}},
		},
	}
}

// loadConfigCall builds the v2 config load from the options a v1 session
// argument amounted to.
func (v *migrateVisitor) loadConfigCall(ctx java.Expression, options []configOption) (java.J, bool) {
	withs := make([]string, len(options))
	for i, o := range options {
		withs[i] = o.with
	}
	t, ok := loadConfigTemplateFor(withs)
	if !ok {
		return nil, false
	}
	values := template.NewMatchResult().Bind(awsCtx, lstutil.SetExprPrefix(ctx, java.EmptySpace))
	for i, o := range options {
		values = values.Bind(awsOpts[i], lstutil.SetExprPrefix(o.value, java.EmptySpace))
	}
	v.flagOptions(options)
	applied := t.Apply(v.Cursor(), values)
	if applied == nil {
		return nil, false
	}
	return v.renamedConfigQualifier(applied), true
}

// flagOptions records the generated helpers and imports loader options name.
func (v *migrateVisitor) flagOptions(options []configOption) {
	for _, o := range options {
		if call, isCall := o.value.(*java.MethodInvocation); isCall && o.with == "WithBaseEndpoint" && call.Name != nil && call.Name.Name == endpointHelper {
			v.needsConfigHelper = true
		}
		if o.with == "WithServiceOptions" {
			if call, isCall := o.value.(*java.MethodInvocation); isCall && call.Name != nil && call.Name.Name == disableSSLHelper {
				v.needsDisableSSL = true
			} else {
				v.needsPathStyle = true
			}
		}
		if o.with == withAssumeRoleCredentialOptions && v.scan.stscredsPkg == "" {
			v.needsStscreds = true
		}
		if o.with == "WithRetryer" {
			v.needsRetry, v.awsUsed, v.needsAws = true, true, true
			v.needsTime = v.needsTime || o.usesTime
		}
	}
}

// credentialsScan finds the credential constructors a config literal consumes.
// v2 has no shared-credentials provider — the loader reads the file itself — so
// such a call carries over only where the rewrite reaches it, and blocks the
// file anywhere else.
type credentialsScan struct {
	visitor.GoVisitor
	scan     *fileScan
	consumed map[java.Expression]bool
}

func (v *credentialsScan) VisitComposite(comp *golang.Composite, p any) java.J {
	// Only a literal a session takes apart becomes loader options; one kept as a
	// config keeps its credentials, rewritten where they stand.
	if options, ok := v.scan.configLiteralOptions(comp); ok && options != nil && consumedAsSessionConfig(v.scan, v.Cursor()) {
		for _, rp := range comp.Elements.Elements {
			kv, isKeyValue := rp.Element.(*golang.KeyValue)
			if !isKeyValue {
				continue
			}
			if key, isIdent := kv.Key.(*java.Identifier); isIdent && key.Name == credentialsField {
				v.consumed[kv.Value.Element] = true
			}
		}
	}
	return v.GoVisitor.VisitComposite(comp, p)
}

// consumesCredentials reports whether a credential constructor is one a config
// literal in the same file takes over.
func (s *fileScan) consumesCredentials(mi *java.MethodInvocation) bool {
	return s.credentialsConsumed[mi]
}

// sharedCredentialsCall rewrites a shared-credentials provider the loader
// options do not reach — one assigned to a config's field rather than set in its
// literal — as the generated helper that loads one.
func (v *migrateVisitor) sharedCredentialsCall(mi *java.MethodInvocation) (java.J, bool) {
	call, isCredentials := qualifiedCall(mi, v.scan.credentialsPkg)
	if !isCredentials || call != "NewSharedCredentials" || v.scan.consumesCredentials(mi) {
		return nil, false
	}
	args := realArgs(mi)
	if len(args) != 2 {
		return nil, false
	}
	ctx := v.contextExpr()
	if ctx == nil {
		return nil, false
	}
	v.needsConfigHelper = true
	c := *mi
	c.Select = nil
	c.Name = &java.Identifier{Name: sharedCredentialsHelper}
	c.MethodType = lstutil.FuncType("", sharedCredentialsHelper, nil)
	c.Arguments = mi.Arguments
	c.Arguments.Elements = []java.RightPadded[java.Expression]{
		{Element: lstutil.SetExprPrefix(ctx, java.EmptySpace)},
		{Element: lstutil.SetExprPrefix(args[0], java.SingleSpace)},
		{Element: lstutil.SetExprPrefix(args[1], java.SingleSpace)},
	}
	return &c, true
}

// v1's stscreds.NewCredentials assumed a role through an STS client built from
// the session, caching the result. v2 spells that as a cache around an
// assume-role provider over an STS client built from the config; both default
// to a 15-minute session and generate the session name.
const stsService = "sts"

// assumeRoleCredentials reports whether mi is stscreds.NewCredentials(sess,
// roleARN), the form with no options to carry.
func (s *fileScan) assumeRoleCredentials(mi *java.MethodInvocation) (sess, role, opts java.Expression, ok bool) {
	call, isStscreds := qualifiedCall(mi, s.stscredsPkg)
	args := realArgs(mi)
	if !isStscreds || call != "NewCredentials" || len(args) < 2 || len(args) > 3 {
		return nil, nil, nil, false
	}
	// A function setting the provider's options sets v2's, where it sets only
	// what they hold.
	if len(args) == 3 {
		if !s.assumeRoleOptionsFunc(args[2]) {
			return nil, nil, nil, false
		}
		opts = args[2]
	}
	return args[0], args[1], opts, true
}

// stsLocal is the name the v2 STS package is reached through.
func (s *fileScan) stsLocal() string {
	if local := s.qualifierFor(stsService); local != "" {
		return local
	}
	return stsService
}

// assumeRoleCall rewrites stscreds.NewCredentials as v2's provider.
func (v *migrateVisitor) assumeRoleCall(mi *java.MethodInvocation) (java.J, bool) {
	sess, role, opts, ok := v.scan.assumeRoleCredentials(mi)
	if !ok {
		return nil, false
	}
	if !v.scan.usesService(stsService) {
		v.needsSTS = true
	}
	v.awsUsed, v.needsAws = true, true
	arg := func(e java.Expression, prefix java.Space) java.RightPadded[java.Expression] {
		return java.RightPadded[java.Expression]{Element: lstutil.SetExprPrefix(e, prefix)}
	}
	call := func(pkg, pkgPath, name string, args ...java.RightPadded[java.Expression]) *java.MethodInvocation {
		return &java.MethodInvocation{
			Select:     &java.RightPadded[java.Expression]{Element: &java.Identifier{Name: pkg, Type: lstutil.NamedType(pkgPath)}},
			Name:       &java.Identifier{Name: name},
			Arguments:  java.Container[java.Expression]{Elements: args},
			MethodType: lstutil.FuncType(pkgPath, name, nil),
		}
	}
	client := call(v.scan.stsLocal(), v2ServicePkg+stsService, "NewFromConfig", arg(sess, java.EmptySpace))
	provider := call(v.scan.stscredsPkg, v2Stscreds, "NewAssumeRoleProvider", arg(client, java.EmptySpace), arg(role, java.SingleSpace))
	if opts != nil {
		provider.Arguments.Elements = append(provider.Arguments.Elements, arg(opts, java.SingleSpace))
	}
	cache := call(v.scan.awsQualifier(), v2Aws, "NewCredentialsCache", arg(provider, java.EmptySpace))
	cache.Prefix = mi.Prefix
	return cache, true
}

// v1 held S3's path-style addressing on the session's config, applying it to
// every S3 client built from it. v2 holds it on the S3 client's options, and its
// config carries service options each client applies when built from it — so a
// generated one sets the addressing style on S3's and leaves the rest alone.
const (
	pathStyleHelper     = "awsCompatS3PathStyle"
	pathStyleHelperFile = "zz_awssdkv2_s3.go"
)

const pathStyleHelperSource = `// Code generated by MigrateAwsSdkGoToV2. DO NOT EDIT.

package %s

import "github.com/aws/aws-sdk-go-v2/service/s3"

// awsCompatS3PathStyle stands in for aws-sdk-go v1's S3ForcePathStyle, which a
// config applied to every S3 client built from it.
func awsCompatS3PathStyle(force bool) func(string, any) {
	return func(_ string, options any) {
		if o, ok := options.(*s3.Options); ok {
			o.UsePathStyle = force
		}
	}
}
`

// pathStyleCall builds the generated service option for a path-style setting.
func pathStyleCall(force java.Expression) java.Expression {
	return &java.MethodInvocation{
		Prefix:     force.GetPrefix(),
		Name:       &java.Identifier{Name: pathStyleHelper},
		Arguments:  java.Container[java.Expression]{Elements: []java.RightPadded[java.Expression]{{Element: lstutil.SetExprPrefix(force, java.EmptySpace)}}},
		MethodType: lstutil.FuncType("", pathStyleHelper, nil),
	}
}

// serviceOptionsLiteral builds `[]func(string, any){option}`.
func serviceOptionsLiteral(option java.Expression) java.Expression {
	return &golang.Composite{
		TypeExpr: &java.ArrayType{ElementType: &java.Identifier{Name: "func(string, any)"}},
		Elements: java.Container[java.Expression]{Elements: []java.RightPadded[java.Expression]{{Element: lstutil.SetExprPrefix(option, java.EmptySpace)}}},
	}
}
