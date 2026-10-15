/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package awssdkv2

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/moderneinc/recipes-go/recipes/internal/lstutil"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/template"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// session.Must panics on a failure the v2 loader returns instead, so one
// statement becomes two: the load, and the panic that Must stood for.
var (
	mustErr           = template.Ident("mustErr")
	mustPanicTemplate = template.StatementTemplate(fmt.Sprintf(`if %s != nil {
	panic(%s)
}`, mustErr, mustErr)).Captures(mustErr).Build()
)

// mustErrName is the error the load binds: err, unless a later statement in the
// block declares err by itself — beside nothing but blanks, as `_, err :=` does
// — which the load's := would leave with no new variable on its left.
func mustErrName(later []java.RightPadded[java.Statement]) string {
	for _, rp := range later {
		switch stmt := rp.Element.(type) {
		case *golang.MultiAssignment:
			if !java.HasMarker[golang.ShortVarDecl](stmt.Markers) {
				continue
			}
			declaresErr, onlyBlanks := false, true
			for _, v := range stmt.Variables {
				id, isIdent := v.Element.(*java.Identifier)
				switch {
				case isIdent && id.Name == "err":
					declaresErr = true
				case !isIdent || id.Name != "_":
					onlyBlanks = false
				}
			}
			if declaresErr && onlyBlanks {
				return "cfgErr"
			}
		case *java.Assignment:
			if id, isIdent := stmt.Variable.(*java.Identifier); isIdent && id.Name == "err" &&
				java.HasMarker[golang.ShortVarDecl](stmt.Markers) {
				return "cfgErr"
			}
		case *java.VariableDeclarations:
			for _, v := range stmt.Variables {
				if v.Element != nil && v.Element.Name != nil && v.Element.Name.Name == "err" {
					return "cfgErr"
				}
			}
		}
	}
	return "err"
}

// mustSession reports whether a statement is `x := session.Must(…)`, returning
// the name bound and the call inside.
func (v *migrateVisitor) mustSession(stmt java.Statement) (name *java.Identifier, inner *java.MethodInvocation, ok bool) {
	assignment, isAssignment := stmt.(*java.Assignment)
	if !isAssignment || !java.HasMarker[golang.ShortVarDecl](assignment.Markers) {
		return nil, nil, false
	}
	target, isIdent := assignment.Variable.(*java.Identifier)
	if !isIdent {
		return nil, nil, false
	}
	must, isCall := assignment.Value.Element.(*java.MethodInvocation)
	if !isCall {
		return nil, nil, false
	}
	if call, isSession := qualifiedCall(must, v.scan.sessionPkg); !isSession || call != "Must" {
		return nil, nil, false
	}
	args := realArgs(must)
	if len(args) != 1 {
		return nil, nil, false
	}
	// The constructor inside was converted on the way in, so what is left is
	// the config load itself. The guard has already refused any Must wrapping
	// something else.
	loaded, isCall := args[0].(*java.MethodInvocation)
	if !isCall {
		return nil, nil, false
	}
	return target, loaded, true
}

// expandMustSession builds the two statements that replace the one: the config
// load, bound to the same name, and the panic Must performed.
func (v *migrateVisitor) expandMustSession(cursor *visitor.Cursor, stmt java.Statement, name *java.Identifier, loaded *java.MethodInvocation, errVar string) ([]java.Statement, bool) {
	errName := &java.Identifier{Prefix: java.SingleSpace, Name: errVar, Type: lstutil.NamedType("error")}
	load := &golang.MultiAssignment{
		ID:      uuid.New(),
		Prefix:  stmt.GetPrefix(),
		Markers: java.MakeMarkers(uuid.New(), []java.Marker{golang.ShortVarDecl{Ident: uuid.New()}}),
		Variables: []java.RightPadded[java.Expression]{
			{Element: &java.Identifier{Name: name.Name, Type: name.Type}},
			{Element: errName},
		},
		Operator: java.LeftPadded[java.Space]{Before: java.SingleSpace},
		Values: []java.RightPadded[java.Expression]{
			{Element: lstutil.SetExprPrefix(loaded, java.SingleSpace)},
		},
	}

	// Applied at the statement being replaced so the template's body picks up
	// that statement's indentation; instantiating it detached would leave the
	// panic at column zero.
	values := template.NewMatchResult().Bind(mustErr, &java.Identifier{Name: errVar, Type: lstutil.NamedType("error")})
	guard := mustPanicTemplate.Apply(visitor.NewCursor(cursor, stmt), values)
	if guard == nil {
		return nil, false
	}
	guardStmt, isStatement := guard.(java.Statement)
	if !isStatement {
		return nil, false
	}
	// A comment above the statement belongs to the load that replaces it, not to
	// the guard that follows; the template picked up the whole prefix.
	guardStmt = lstutil.SetStmtPrefix(guardStmt, java.MakeSpace(nil, guardStmt.GetPrefix().Whitespace()))
	return []java.Statement{load, guardStmt}, true
}

// mustExpression rewrites a session.Must the block cannot expand into
// statements — one returned, passed on, or assigned other than by := — as the
// generated loader, which panics on a load failure as Must did. A config the
// caller already held cannot fail to load, so Must around it is just that
// config.
func (v *migrateVisitor) mustExpression(mi *java.MethodInvocation, inner java.Expression) (java.Expression, bool) {
	if v.expandableMust() {
		return nil, false
	}
	loaded, isCall := inner.(*java.MethodInvocation)
	if !isCall || loaded.Name == nil {
		return nil, false
	}
	switch {
	case loaded.Name.Name == sessionConfigHelper && loaded.Select == nil:
		if args := realArgs(loaded); len(args) == 1 {
			v.carriedConfigs--
			return lstutil.SetExprPrefix(args[0], mi.Prefix), true
		}
	case loaded.Name.Name == "LoadDefaultConfig" && loaded.Select != nil:
		v.needsConfigHelper = true
		return lstutil.SetExprPrefix(asCompatConfig(loaded), mi.Prefix), true
	}
	return nil, false
}

// expandableMust reports whether the call being visited is the value of an `x := session.Must(…)`
// statement, which VisitBlock expands into the load and the panic.
func (v *migrateVisitor) expandableMust() bool {
	cursor := v.Cursor()
	for cursor != nil && cursor.Value() == nil {
		cursor = cursor.Parent()
	}
	if cursor == nil {
		return false
	}
	assignment, isAssignment := parentTree(cursor).(*java.Assignment)
	if !isAssignment {
		return false
	}
	_, isIdent := assignment.Variable.(*java.Identifier)
	if !isIdent || !java.HasMarker[golang.ShortVarDecl](assignment.Markers) {
		return false
	}
	for c := cursor.Parent(); c != nil; c = c.Parent() {
		if c.Value() == java.Tree(assignment) {
			_, isBlock := parentTree(c).(*java.Block)
			return isBlock
		}
	}
	return false
}

// v1's session.New built a session in an expression, deferring any load failure
// to the first call. v2's loader returns that failure, and an expression has
// nowhere to return it from, so the load moves into a generated helper that
// panics as session.Must did.
const (
	configHelper            = "awsCompatConfig"
	endpointHelper          = "awsCompatEndpoint"
	sharedCredentialsHelper = "awsCompatSharedCredentials"
	sessionConfigHelper     = "awsCompatSessionConfig"
	configFileName          = "zz_awssdkv2_config.go"
)

const configHelperSource = `// Code generated by MigrateAwsSdkGoToV2. DO NOT EDIT.

package %s

import (
	"context"
	"regexp"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
)

// awsCompatConfig stands in for aws-sdk-go v1's session.New, which built a
// session in an expression and deferred a load failure to the first call. v2
// returns that failure, and an expression cannot, so it panics the way
// session.Must did.
func awsCompatConfig(ctx context.Context, optFns ...func(*config.LoadOptions) error) aws.Config {
	cfg, err := config.LoadDefaultConfig(ctx, optFns...)
	if err != nil {
		panic(err)
	}
	return cfg
}

// awsCompatSharedCredentials stands in for aws-sdk-go v1's
// credentials.NewSharedCredentials. v2 has no shared-credentials provider: the
// loader reads the file and the profile itself, so the provider is taken off a
// config loaded for exactly that.
func awsCompatSharedCredentials(ctx context.Context, file, profile string) aws.CredentialsProvider {
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithSharedCredentialsFiles([]string{file}),
		config.WithSharedConfigProfile(profile))
	if err != nil {
		panic(err)
	}
	return cfg.Credentials
}

// awsCompatSessionConfig stands in for a session built from a config the caller
// already had. v2's aws.Config is that config, so there is nothing left to
// build and nothing left to fail; the error is kept so the call site reads as
// it did.
func awsCompatSessionConfig(cfg aws.Config) (aws.Config, error) {
	return cfg, nil
}

// awsCompatEndpoint gives an endpoint written without a scheme the https one
// aws-sdk-go v1 added; v2 takes the URL as it is given. Empty stays empty,
// which both read as no endpoint at all.
func awsCompatEndpoint(endpoint string) string {
	if endpoint == "" || awsCompatSchemeRE.MatchString(endpoint) {
		return endpoint
	}
	return "https://" + endpoint
}

var awsCompatSchemeRE = regexp.MustCompile("^([^:]+)://")
`

// sessionNewConfig rewrites session.New as the generated loader, which takes
// the same options the config load does.
func (v *migrateVisitor) sessionNewConfig(mi *java.MethodInvocation) java.J {
	applied, isCall := v.loadConfig(mi).(*java.MethodInvocation)
	if !isCall || applied == mi || applied.Select == nil {
		return mi
	}
	v.needsConfigHelper = true
	return lstutil.SetExprPrefix(asCompatConfig(applied), mi.Prefix)
}

// compatConfigCall builds the generated loader's call from a list of options.
func (v *migrateVisitor) compatConfigCall(ctx java.Expression, options []configOption) (java.Expression, bool) {
	loaded, built := v.loadConfigCall(ctx, options)
	if !built {
		return nil, false
	}
	call, isCall := loaded.(*java.MethodInvocation)
	if !isCall || call.Select == nil {
		return nil, false
	}
	v.needsConfigHelper = true
	return asCompatConfig(call), true
}

// asCompatConfig turns a config.LoadDefaultConfig call into the generated
// loader's, which answers with the config alone and panics as session.Must did.
func asCompatConfig(call *java.MethodInvocation) java.Expression {
	c := *call
	c.Select = nil
	c.Name = &java.Identifier{Name: configHelper}
	c.MethodType = lstutil.FuncType("", configHelper, nil)
	return &c
}
