/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling

import (
	"github.com/google/uuid"
	"github.com/moderneinc/recipes-go/diagnostic"
	"github.com/moderneinc/recipes-go/recipes/internal/lstutil"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/matcher"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	recipegolang "github.com/openrewrite/rewrite/rewrite-go/pkg/recipe/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// UseErrorsAs transforms direct type assertions on errors like
// `if myErr, ok := err.(*MyError); ok { ... }` into
// `var myErr *MyError; if errors.As(err, &myErr) { ... }`.
// This correctly handles wrapped errors via the errors package.
type UseErrorsAs struct {
	recipe.Base
}

func (r *UseErrorsAs) Name() string {
	return "org.openrewrite.golang.codequality.UseErrorsAs"
}
func (r *UseErrorsAs) DisplayName() string {
	return "Use errors.As"
}
func (r *UseErrorsAs) Description() string {
	return "Replace `if myErr, ok := err.(*MyError); ok { ... }` with `var myErr *MyError; if errors.As(err, &myErr) { ... }` for correct wrapped error handling."
}
func (r *UseErrorsAs) Tags() []string { return []string{"errorhandling", "lint"} }

func (r *UseErrorsAs) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{
		{DiagnosticID: "errorlint", Tool: diagnostic.GolangciLint, HasFix: true},
	}
}

func (r *UseErrorsAs) Editor() recipe.TreeVisitor {
	return visitor.Init(&useErrorsAsVisitor{})
}

type useErrorsAsVisitor struct {
	visitor.GoVisitor
}

// VisitBlock finds if-statements with init of the form
// `myErr, ok := err.(*MyError); ok` and transforms them into
// `var myErr *MyError` + `if errors.As(err, &myErr) { ... }`.
func (v *useErrorsAsVisitor) VisitBlock(block *java.Block, p any) java.J {
	block = v.GoVisitor.VisitBlock(block, p).(*java.Block)

	changed := false
	var newStmts []java.RightPadded[java.Statement]

	for _, rp := range block.Statements {
		// `if init; cond` is wrapped in a golang.StatementWithInit; the comma-ok
		// type assertion lives in its init clause.
		swi, ok := rp.Element.(*golang.StatementWithInit)
		if !ok {
			newStmts = append(newStmts, rp)
			continue
		}

		varName, typeExpr, errExpr := matchCommaOkTypeAssert(swi)
		if varName == "" {
			newStmts = append(newStmts, rp)
			continue
		}

		changed = true

		ifStmt := swi.Statement.(*java.If)

		// Build: var myErr *MyError
		varDecl := buildVarDecl(varName, typeExpr, swi.Prefix)

		// Build: if errors.As(err, &myErr) { ... }
		newIf := buildErrorsAsIf(ifStmt, swi.Prefix, errExpr, varName)

		newStmts = append(newStmts,
			java.RightPadded[java.Statement]{Element: varDecl},
			java.RightPadded[java.Statement]{Element: newIf, After: rp.After, Markers: rp.Markers},
		)
	}

	if !changed {
		return block
	}

	// The rewrite introduces a reference to the `errors` package; ensure it is imported.
	recipegolang.MaybeAddImport(v, "errors", nil, false)

	return block.WithStatements(newStmts)
}

// matchCommaOkTypeAssert checks if an If statement has init of the form:
//
//	myErr, ok := err.(*MyError); ok
//
// Returns (varName, typeExpr, errExpr) or ("", nil, nil) if no match.
func matchCommaOkTypeAssert(swi *golang.StatementWithInit) (string, java.Expression, java.Expression) {
	ifStmt, ok := swi.Statement.(*java.If)
	if !ok {
		return "", nil, nil
	}

	ma, ok := swi.Init.Element.(*golang.MultiAssignment)
	if !ok {
		return "", nil, nil
	}

	// Must be a short var decl (:=)
	if !java.HasMarker[golang.ShortVarDecl](ma.Markers) {
		return "", nil, nil
	}

	// Must have exactly 2 variables and 1 value
	if len(ma.Variables) != 2 || len(ma.Values) != 1 {
		return "", nil, nil
	}

	// The value must be a type assertion
	ta, ok := ma.Values[0].Element.(*golang.TypeAssertion)
	if !ok {
		return "", nil, nil
	}

	// The condition must be the "ok" identifier
	if ifStmt.Condition == nil {
		return "", nil, nil
	}
	condIdent, ok := ifStmt.Condition.Tree.Element.(*java.Identifier)
	if !ok || condIdent.Name != "ok" {
		return "", nil, nil
	}

	// Second variable must be "ok"
	okIdent, ok := ma.Variables[1].Element.(*java.Identifier)
	if !ok || okIdent.Name != "ok" {
		return "", nil, nil
	}

	// First variable is the target name (e.g., myErr)
	targetIdent, ok := ma.Variables[0].Element.(*java.Identifier)
	if !ok {
		return "", nil, nil
	}

	// The expression being asserted must be an error.
	// Check type info first; fall back to name heuristic.
	if !looksLikeError(ta.Left.Element) {
		return "", nil, nil
	}

	// Extract the type from the type assertion (inside the ControlParentheses)
	if ta.AssertedType == nil {
		return "", nil, nil
	}
	typeExpr := ta.AssertedType.Tree.Element

	return targetIdent.Name, typeExpr, ta.Left.Element
}

// Reports whether the expression is assignable to error, deciding from the
// resolved type when present and otherwise from the "err" naming convention.
func looksLikeError(expr java.Expression) bool {
	ident, ok := expr.(*java.Identifier)
	if !ok {
		return false
	}
	// A resolved type is decisive: an `any`/`interface{}` value is not assignable
	// to error, so errors.As would not compile.
	if ident.Type != nil {
		return matcher.IsAssignableTo(ident.Type, "error")
	}
	// Fall back to name convention only when the type is unresolved.
	return ident.Name == "err"
}

// buildVarDecl constructs: var varName typeExpr
func buildVarDecl(varName string, typeExpr java.Expression, prefix java.Space) *java.VariableDeclarations {
	return &java.VariableDeclarations{
		ID:     uuid.New(),
		Prefix: prefix,
		Markers: java.Markers{
			ID:      uuid.New(),
			Entries: []java.Marker{golang.VarKeyword{Ident: uuid.New()}},
		},
		TypeExpr: setExprPrefix(typeExpr, java.SingleSpace),
		Variables: []java.RightPadded[*java.VariableDeclarator]{
			{
				Element: &java.VariableDeclarator{
					ID: uuid.New(),
					Name: &java.Identifier{
						ID:     uuid.New(),
						Prefix: java.SingleSpace,
						Name:   varName,
					},
				},
			},
		},
	}
}

// buildErrorsAsIf constructs: if errors.As(errExpr, &varName) { <original body> }
func buildErrorsAsIf(origIf *java.If, prefix java.Space, errExpr java.Expression, varName string) *java.If {
	errorsAsCall := &java.MethodInvocation{
		ID: uuid.New(),
		Select: &java.RightPadded[java.Expression]{
			Element: &java.Identifier{
				ID:     uuid.New(),
				Prefix: java.SingleSpace,
				Name:   "errors",
				Type:   lstutil.NamedType("errors"),
			},
		},
		Name: &java.Identifier{
			ID:   uuid.New(),
			Name: "As",
		},
		MethodType: lstutil.FuncType("errors", "As", lstutil.BoolType),
		Arguments: java.Container[java.Expression]{
			Elements: []java.RightPadded[java.Expression]{
				{
					Element: setExprPrefix(errExpr, java.EmptySpace),
				},
				{
					Element: &golang.Unary{
						ID:       uuid.New(),
						Prefix:   java.SingleSpace,
						Operator: java.LeftPadded[golang.UnaryOperator]{Element: golang.AddressOf},
						Expression: &java.Identifier{
							ID:   uuid.New(),
							Name: varName,
						},
					},
				},
			},
		},
	}

	return &java.If{
		ID:     uuid.New(),
		Prefix: prefix,
		Condition: &java.ControlParentheses{
			ID:   uuid.New(),
			Tree: java.RightPadded[java.Expression]{Element: errorsAsCall},
		},
		ThenPart: origIf.ThenPart,
		ElsePart: origIf.ElsePart,
	}
}

// setExprPrefix sets the prefix on an expression node.
func setExprPrefix(expr java.Expression, prefix java.Space) java.Expression {
	switch e := expr.(type) {
	case *java.Identifier:
		return e.WithPrefix(prefix)
	case *java.Unary:
		return e.WithPrefix(prefix)
	case *golang.Unary:
		return e.WithPrefix(prefix)
	case *java.FieldAccess:
		c := *e
		c.Target = setExprPrefix(c.Target, prefix).(java.Expression)
		return &c
	default:
		return expr
	}
}
