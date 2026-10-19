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

// A v1 config built up through its With setters in a local — some applied only
// under a condition — took effect when the session was made, which is also when
// DisableSSL decided the endpoint's scheme. v2's config has no setters, so the
// local becomes a generated builder answering the same setters, resolved into
// loader options when the config is loaded. The call sites keep their shape.

const (
	builderType   = "awsCompatConfigBuilder"
	builderHelper = "awsCompatNewConfig"
	builderFile   = "zz_awssdkv2_configbuilder.go"
	builderLoad   = "load"
)

// builderSetters are the setters the builder answers.
var builderSetters = map[string]bool{
	"WithRegion": true, "WithEndpoint": true, "WithDisableSSL": true,
	"WithS3ForcePathStyle": true, "WithMaxRetries": true, "WithCredentials": true,
	"WithHTTPClient": true,
}

// chainOnto returns what a chain of builder setters is called on.
func chainOnto(expr java.Expression) (java.Expression, bool) {
	for {
		mi, isCall := expr.(*java.MethodInvocation)
		if !isCall || mi.Name == nil || mi.Select == nil || !builderSetters[mi.Name.Name] || len(realArgs(mi)) != 1 {
			return expr, true
		}
		expr = mi.Select.Element
	}
}

// configBuilderLocals finds the locals declared from an aws.NewConfig chain
// whose every other use is setting more on it or making the session from it.
func (s *fileScan) configBuilderLocals(cu *golang.CompilationUnit) map[string]bool {
	scan := visitor.Init(&builderScan{scan: s, accepted: map[string]int{}, declared: map[string]bool{}})
	scan.Visit(cu, nil)
	out := map[string]bool{}
	for name := range scan.declared {
		// A local set once and handed on folds into the load as it stands.
		if s.droppableConfigVar(name) {
			continue
		}
		if scan.accepted[name] == s.configUses[name] {
			out[name] = true
		}
	}
	return out
}

type builderScan struct {
	visitor.GoVisitor
	scan     *fileScan
	accepted map[string]int
	declared map[string]bool
}

func (v *builderScan) VisitAssignment(a *java.Assignment, p any) java.J {
	target, isIdent := a.Variable.(*java.Identifier)
	if !isIdent {
		return v.GoVisitor.VisitAssignment(a, p)
	}
	if java.HasMarker[golang.ShortVarDecl](a.Markers) {
		root, ok := chainOnto(a.Value.Element)
		if call, isCall := root.(*java.MethodInvocation); ok && isCall {
			if helper, isAws := qualifiedCall(call, v.scan.awsPkg); isAws && helper == "NewConfig" && len(realArgs(call)) == 0 {
				v.declared[target.Name] = true
				v.accepted[target.Name]++
			}
		}
	} else if root, ok := chainOnto(a.Value.Element); ok {
		if id, isRoot := root.(*java.Identifier); isRoot && id.Name == target.Name && a.Value.Element != root {
			v.accepted[target.Name] += 2
		}
	}
	return v.GoVisitor.VisitAssignment(a, p)
}

// A session.Options literal carrying the config by its Config key takes it
// whole, as the loader's options value.
func (v *builderScan) VisitComposite(comp *golang.Composite, p any) java.J {
	if _, isOptions := v.scan.sessionOptionsLiteral(comp); isOptions {
		if local, ok := builderCarried(comp); ok {
			v.accepted[local.Name]++
		}
	}
	return v.GoVisitor.VisitComposite(comp, p)
}

// builderCarried returns the local a session.Options literal's Config names,
// dereferenced or not, where its other keys need nothing in v2.
func builderCarried(comp *golang.Composite) (*java.Identifier, bool) {
	var local *java.Identifier
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
		switch key.Name {
		case sessionConfig:
			value := kv.Value.Element
			if unary, isUnary := value.(*golang.Unary); isUnary && unary.Operator.Element == golang.Indirection {
				value = unary.Expression
			}
			id, isLocal := value.(*java.Identifier)
			if !isLocal {
				return nil, false
			}
			local = id
		case sharedConfigState:
			// Checked as for any options literal: only enabling it converts.
		default:
			return nil, false
		}
	}
	return local, local != nil
}

func (v *builderScan) VisitMethodInvocation(mi *java.MethodInvocation, p any) java.J {
	if call, isSession := qualifiedCall(mi, v.scan.sessionPkg); isSession && call == "NewSession" {
		if args := realArgs(mi); len(args) == 1 {
			if id, isIdent := args[0].(*java.Identifier); isIdent {
				v.accepted[id.Name]++
			}
		}
	}
	return v.GoVisitor.VisitMethodInvocation(mi, p)
}

// builderHead reports whether the cursor's aws.NewConfig() heads the chain a
// builder local is declared from.
func (s *fileScan) builderHead(cursor *visitor.Cursor) bool {
	c := cursor
	for {
		parent := c.Parent()
		for parent != nil && parent.Value() == nil {
			parent = parent.Parent()
		}
		if parent == nil {
			return false
		}
		switch p := parent.Value().(type) {
		case *java.MethodInvocation:
			if p.Select == nil || java.Tree(p.Select.Element) != c.Value() {
				return false
			}
			c = parent
			continue
		case *java.Assignment:
			target, isIdent := p.Variable.(*java.Identifier)
			return isIdent && java.HasMarker[golang.ShortVarDecl](p.Markers) && s.builderVars[target.Name]
		}
		return false
	}
}

// builderVar reports whether expr names a builder local.
func (s *fileScan) builderVar(expr java.Expression) (*java.Identifier, bool) {
	id, isIdent := expr.(*java.Identifier)
	return id, isIdent && s.builderVars[id.Name]
}

func builderHeadCall(prefix java.Space) java.Expression {
	return &java.MethodInvocation{
		Prefix:     prefix,
		Name:       &java.Identifier{Name: builderHelper},
		MethodType: lstutil.FuncType("", builderHelper, nil),
	}
}

// builderLoadCall loads the config a builder holds: `cfg.load(ctx)`.
func builderLoadCall(prefix java.Space, local *java.Identifier, ctx java.Expression) java.Expression {
	return &java.MethodInvocation{
		Prefix:     prefix,
		Select:     &java.RightPadded[java.Expression]{Element: lstutil.SetExprPrefix(local, java.EmptySpace)},
		Name:       &java.Identifier{Name: builderLoad},
		Arguments:  java.Container[java.Expression]{Elements: []java.RightPadded[java.Expression]{{Element: lstutil.SetExprPrefix(ctx, java.EmptySpace)}}},
		MethodType: lstutil.FuncType("", builderLoad, nil),
	}
}

const builderSource = `// Code generated by MigrateAwsSdkGoToV2. DO NOT EDIT.

package %s

import (
	"context"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
)

// awsCompatConfigBuilder stands in for an aws-sdk-go v1 aws.Config built up
// through its With setters. As in v1, the settings take effect when the config
// is loaded, which is when DisableSSL decides the endpoint's scheme.
type awsCompatConfigBuilder struct {
	region, endpoint      *string
	disableSSL, pathStyle *bool
	maxRetries            *int
	credentials           aws.CredentialsProvider
	httpClient            *http.Client
}

func awsCompatNewConfig() *awsCompatConfigBuilder { return &awsCompatConfigBuilder{} }

func (c *awsCompatConfigBuilder) WithRegion(region string) *awsCompatConfigBuilder {
	c.region = &region
	return c
}

func (c *awsCompatConfigBuilder) WithEndpoint(endpoint string) *awsCompatConfigBuilder {
	c.endpoint = &endpoint
	return c
}

func (c *awsCompatConfigBuilder) WithDisableSSL(disable bool) *awsCompatConfigBuilder {
	c.disableSSL = &disable
	return c
}

func (c *awsCompatConfigBuilder) WithS3ForcePathStyle(force bool) *awsCompatConfigBuilder {
	c.pathStyle = &force
	return c
}

func (c *awsCompatConfigBuilder) WithMaxRetries(max int) *awsCompatConfigBuilder {
	c.maxRetries = &max
	return c
}

func (c *awsCompatConfigBuilder) WithCredentials(credentials aws.CredentialsProvider) *awsCompatConfigBuilder {
	c.credentials = credentials
	return c
}

func (c *awsCompatConfigBuilder) WithHTTPClient(client *http.Client) *awsCompatConfigBuilder {
	c.httpClient = client
	return c
}

// load loads the v2 config the settings amount to.
func (c *awsCompatConfigBuilder) load(ctx context.Context) (aws.Config, error) {
	return awsconfig.LoadDefaultConfig(ctx, c.loadOptions()...)
}

// options answers the settings as the loader's options value, for a v1
// session.Options that carried the config.
func (c *awsCompatConfigBuilder) options() awsconfig.LoadOptions {
	var o awsconfig.LoadOptions
	for _, opt := range c.loadOptions() {
		_ = opt(&o)
	}
	return o
}

func (c *awsCompatConfigBuilder) loadOptions() []func(*awsconfig.LoadOptions) error {
	var opts []func(*awsconfig.LoadOptions) error
	disableSSL := c.disableSSL != nil && *c.disableSSL
	if c.region != nil {
		opts = append(opts, awsconfig.WithRegion(*c.region))
	}
	if c.endpoint != nil && *c.endpoint != "" {
		opts = append(opts, awsconfig.WithBaseEndpoint(awsCompatEndpointScheme(*c.endpoint, disableSSL)))
	}
	if disableSSL {
		opts = append(opts, awsconfig.WithServiceOptions(awsCompatDisableSSL(true)))
	}
	if c.pathStyle != nil {
		opts = append(opts, awsconfig.WithServiceOptions(awsCompatS3PathStyle(*c.pathStyle)))
	}
	// v1 counted retries, v2 attempts; a negative count asked for the default.
	if c.maxRetries != nil && *c.maxRetries >= 0 {
		opts = append(opts, awsconfig.WithRetryMaxAttempts(*c.maxRetries+1))
	}
	if c.credentials != nil {
		opts = append(opts, awsconfig.WithCredentialsProvider(c.credentials))
	}
	if c.httpClient != nil {
		opts = append(opts, awsconfig.WithHTTPClient(c.httpClient))
	}
	return opts
}
`

// configFields map a loader option to the v2 aws.Config field it sets, for a
// NewConfig chain kept as the config a client merges over its session's.
var configFields = map[string]string{
	"WithRegion":            "Region",
	"WithBaseEndpoint":      "BaseEndpoint",
	"WithRetryMaxAttempts":  "RetryMaxAttempts",
	"WithHTTPClient":        "HTTPClient",
	withCredentialsProvider: "Credentials",
	"WithRetryer":           "Retryer",
}

// keptChainLiteral builds the v2 config a kept NewConfig chain amounts to.
func (s *fileScan) keptChainLiteral(options []configOption) (java.Expression, bool) {
	var fields, serviceOptions []string
	values := map[string]java.Expression{}
	for i, o := range options {
		placeholder := fmt.Sprintf("awsConfigOption%d", i)
		values[placeholder] = o.value
		if o.with == "WithServiceOptions" {
			serviceOptions = append(serviceOptions, placeholder)
			continue
		}
		field, ok := configFields[o.with]
		if !ok {
			return nil, false
		}
		// v2 holds the endpoint as a pointer, which the loader option did not.
		if field == "BaseEndpoint" {
			placeholder = fmt.Sprintf("%s.String(%s)", s.awsQualifier(), placeholder)
		}
		fields = append(fields, fmt.Sprintf("\t%s: %s,\n", field, placeholder))
	}
	if len(serviceOptions) > 0 {
		fields = append(fields, fmt.Sprintf("\tServiceOptions: []func(string, any){%s},\n", strings.Join(serviceOptions, ", ")))
	}
	source := fmt.Sprintf("package p\n\nvar _ = &%s.Config{\n%s}\n", s.awsQualifier(), strings.Join(fields, ""))
	cu, err := parser.NewGoParser().Parse("config.go", source)
	if err != nil {
		return nil, false
	}
	lift := visitor.Init(&compositeLift{})
	lift.Visit(cu, nil)
	if lift.literal == nil {
		return nil, false
	}
	realign(lift.literal.Elements.Elements)
	typed := visitor.Init(&qualifiedCallTyping{paths: map[string]string{s.awsQualifier(): v2Aws}}).Visit(lift.literal, nil).(java.Expression)
	var literal java.Expression = &golang.Unary{Operator: java.LeftPadded[golang.UnaryOperator]{Element: golang.AddressOf}, Expression: typed}
	for name, value := range values {
		bound, ok := visitor.Init(&placeholderBind{name: name, value: value}).Visit(literal, nil).(java.Expression)
		if !ok {
			return nil, false
		}
		literal = bound
	}
	return literal, true
}

// keptChainDeclaration rewrites the declaration of a kept config local built
// as a NewConfig chain.
func (v *migrateVisitor) keptChainDeclaration(a *java.Assignment) (*java.Assignment, bool) {
	target, isIdent := a.Variable.(*java.Identifier)
	if !isIdent || !java.HasMarker[golang.ShortVarDecl](a.Markers) || !v.scan.configKept[target.Name] {
		return nil, false
	}
	value := a.Value.Element
	if unary, isUnary := value.(*golang.Unary); isUnary {
		value = unary.Expression
	}
	options, isChain := v.scan.configChainOptions(value)
	if !isChain {
		return nil, false
	}
	literal, built := v.scan.keptChainLiteral(options)
	if !built {
		return nil, false
	}
	v.flagOptions(options)
	v.awsUsed, v.needsAws = true, true
	c := *a
	c.Value = java.LeftPadded[java.Expression]{Before: a.Value.Before, Element: lstutil.SetExprPrefix(literal, a.Value.Element.GetPrefix()), Markers: a.Value.Markers}
	return &c, true
}
