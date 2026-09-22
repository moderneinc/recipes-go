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
}

// configFieldOptions map an aws.Config field to the loader option that replaces
// it. A field absent from this table has no counterpart the recipe writes.
var configFieldOptions = map[string]string{
	"Region":     "WithRegion",
	"Endpoint":   "WithBaseEndpoint",
	"MaxRetries": "WithRetryMaxAttempts",
	"HTTPClient": "WithHTTPClient",
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

// configLiteralOptions reads an aws.Config literal, returning the loader options
// it amounts to. The literal may be written with or without its address taken.
func (s *fileScan) configLiteralOptions(expr java.Expression) ([]configOption, bool) {
	if unary, isUnary := expr.(*golang.Unary); isUnary {
		expr = unary.Expression
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
		converted, ok := s.configFieldOption(key.Name, kv.Value.Element)
		if !ok {
			return nil, false
		}
		options = append(options, converted...)
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
	return []configOption{{with: with, value: inner}}, true
}

// credentialsOptions maps v1's credential constructors to the loader options
// that stand in for them.
func (s *fileScan) credentialsOptions(value java.Expression) ([]configOption, bool) {
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
	applied := t.Apply(v.Cursor(), values)
	if applied == nil {
		return nil, false
	}
	v.configUsed = true
	return v.renamedConfigQualifier(applied), true
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
	if options, ok := v.scan.configLiteralOptions(comp); ok && options != nil {
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
