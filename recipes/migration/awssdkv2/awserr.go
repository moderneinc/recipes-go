/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package awssdkv2

import (
	"github.com/google/uuid"
	"github.com/moderneinc/recipes-go/recipes/internal/lstutil"
	"github.com/moderneinc/recipes-go/recipes/migration/awssdkv2/awsmanifest"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

const (
	v1Awserr     = v1Aws + "/awserr"
	smithyGo     = "github.com/aws/smithy-go"
	errorsPkg    = "errors"
	smithyAPIErr = "APIError"
)

// awserrMethodRenames map the v1 error interface's accessors to smithy's.
var awserrMethodRenames = map[string]string{
	"Code":    "ErrorCode",
	"Message": "ErrorMessage",
}

// awserrSupported are the awserr names with a mechanical v2 form. The
// constructors and the batched shapes have none, and block their file.
var awserrSupported = map[string]bool{"Error": true}

// awserrAssertion reports whether an if's init asserts an error to
// awserr.Error, returning the bound name, the ok name and the error expression.
// This is the shape every caller writes: `if aerr, ok := err.(awserr.Error); …`.
func (s *fileScan) awserrAssertion(swi *golang.StatementWithInit) (bound, okName *java.Identifier, errExpr java.Expression, ok bool) {
	if s.awserrPkg == "" {
		return nil, nil, nil, false
	}
	assignment, isMulti := swi.Init.Element.(*golang.MultiAssignment)
	if !isMulti || len(assignment.Variables) != 2 || len(assignment.Values) != 1 {
		return nil, nil, nil, false
	}
	bound, isIdent := assignment.Variables[0].Element.(*java.Identifier)
	if !isIdent {
		return nil, nil, nil, false
	}
	okName, isIdent = assignment.Variables[1].Element.(*java.Identifier)
	if !isIdent {
		return nil, nil, nil, false
	}
	assertion, isAssertion := assignment.Values[0].Element.(*golang.TypeAssertion)
	if !isAssertion || assertion.AssertedType == nil {
		return nil, nil, nil, false
	}
	fa, isField := assertion.AssertedType.Tree.Element.(*java.FieldAccess)
	if !isField || fa.Name.Element == nil || fa.Name.Element.Name != "Error" {
		return nil, nil, nil, false
	}
	pkg, isIdent := fa.Target.(*java.Identifier)
	if !isIdent || pkg.Name != s.awserrPkg {
		return nil, nil, nil, false
	}
	return bound, okName, assertion.Left.Element, true
}

// expandAwserrAssertion turns the v1 assertion into smithy's errors.As form:
// the interface is declared first, then matched, since errors.As takes a
// pointer to it.
func expandAwserrAssertion(cursor *visitor.Cursor, swi *golang.StatementWithInit, bound, okName *java.Identifier, errExpr java.Expression, errorsLocal string) ([]java.Statement, bool) {
	iff, isIf := swi.Statement.(*java.If)
	if !isIf || iff.Condition == nil {
		return nil, false
	}

	declaration := &java.VariableDeclarations{
		ID:       uuid.New(),
		Prefix:   swi.Prefix,
		Markers:  java.Markers{ID: uuid.New()},
		TypeExpr: smithyAPIErrorType(),
		Variables: []java.RightPadded[*java.VariableDeclarator]{
			{Element: &java.VariableDeclarator{
				ID:   uuid.New(),
				Name: &java.Identifier{Prefix: java.SingleSpace, Name: bound.Name, Type: lstutil.NamedType(smithyGo + "." + smithyAPIErr)},
			}},
		},
	}
	declaration.Markers = java.Markers{ID: uuid.New(), Entries: []java.Marker{golang.VarKeyword{Ident: uuid.New()}}}

	// The `ok` the assertion produced becomes the errors.As call itself.
	matched := errorsAsCall(errExpr, bound, errorsLocal)
	condition := replaceIdentifier(iff.Condition.Tree.Element, okName.Name, matched)

	rewrittenIf := *iff
	rewrittenIf.Prefix = swi.Prefix
	control := *iff.Condition
	control.Tree = java.RightPadded[java.Expression]{Element: condition, After: iff.Condition.Tree.After}
	rewrittenIf.Condition = &control

	return []java.Statement{declaration, &rewrittenIf}, true
}

// smithyAPIErrorType names the interface v2 reports an API failure through.
func smithyAPIErrorType() java.Expression {
	return &java.FieldAccess{
		ID:     uuid.New(),
		Prefix: java.SingleSpace,
		Target: &java.Identifier{ID: uuid.New(), Name: "smithy", Type: lstutil.NamedType(smithyGo)},
		Name:   java.LeftPadded[*java.Identifier]{Element: &java.Identifier{ID: uuid.New(), Name: smithyAPIErr}},
		Type:   lstutil.NamedType(smithyGo + "." + smithyAPIErr),
	}
}

// errorsAsCall builds `errors.As(err, &bound)`.
func errorsAsCall(errExpr java.Expression, bound *java.Identifier, local string) java.Expression {
	address := &golang.Unary{
		ID:         uuid.New(),
		Operator:   java.LeftPadded[golang.UnaryOperator]{Element: golang.AddressOf},
		Expression: &java.Identifier{ID: uuid.New(), Name: bound.Name},
	}
	return &java.MethodInvocation{
		ID:     uuid.New(),
		Select: &java.RightPadded[java.Expression]{Element: &java.Identifier{ID: uuid.New(), Name: local, Type: lstutil.NamedType(errorsPkg)}},
		Name:   &java.Identifier{ID: uuid.New(), Name: "As"},
		Arguments: java.Container[java.Expression]{Elements: []java.RightPadded[java.Expression]{
			{Element: lstutil.SetExprPrefix(errExpr, java.EmptySpace)},
			{Element: lstutil.SetExprPrefix(address, java.SingleSpace)},
		}},
		MethodType: lstutil.FuncType(errorsPkg, "As", nil),
	}
}

// replaceIdentifier swaps every reference to name in expr for replacement,
// which is how the assertion's `ok` becomes the errors.As call that now decides
// the branch.
func replaceIdentifier(expr java.Expression, name string, replacement java.Expression) java.Expression {
	v := visitor.Init(&identifierSwap{name: name, replacement: replacement})
	if out, ok := v.Visit(expr, nil).(java.Expression); ok {
		return out
	}
	return expr
}

type identifierSwap struct {
	visitor.GoVisitor
	name        string
	replacement java.Expression
}

func (v *identifierSwap) VisitIdentifier(id *java.Identifier, p any) java.J {
	if id.Name != v.name {
		return v.GoVisitor.VisitIdentifier(id, p)
	}
	return lstutil.SetExprPrefix(v.replacement, id.Prefix)
}

// errCodeLiteral rewrites a v1 ErrCode constant to the wire string it held,
// which is what a v2 caller compares smithy's ErrorCode against.
func errCodeLiteral(service, name string, prefix java.Space) (java.Expression, bool) {
	value, ok := awsmanifest.ErrCodeValue(service, name)
	if !ok {
		return nil, false
	}
	return &java.Literal{
		ID:      uuid.New(),
		Prefix:  prefix,
		Value:   value,
		Source:  `"` + value + `"`,
		Type:    &java.JavaTypePrimitive{Keyword: "String"},
		Markers: java.Markers{ID: uuid.New()},
	}, true
}

// awserrBoundNames collects the names an awserr.Error assertion binds anywhere
// in the file. The accessors on them are renamed while visiting an if body the
// expansion only reaches on the way out, so the names are gathered up front.
func awserrBoundNames(cu *golang.CompilationUnit, scan *fileScan) map[string]bool {
	found := map[string]bool{}
	if scan.awserrPkg == "" {
		return found
	}
	collect := visitor.Init(&awserrBoundScan{scan: scan, found: found})
	collect.Visit(cu, nil)
	return found
}

type awserrBoundScan struct {
	visitor.GoVisitor
	scan  *fileScan
	found map[string]bool
}

func (v *awserrBoundScan) VisitStatementWithInit(swi *golang.StatementWithInit, p any) java.J {
	if bound, _, _, ok := v.scan.awserrAssertion(swi); ok {
		v.found[bound.Name] = true
	}
	return v.GoVisitor.VisitStatementWithInit(swi, p)
}
