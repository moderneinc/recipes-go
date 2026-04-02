/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// PreferErrorsIsForFieldAccess replaces `err == sentinel` with `errors.Is(err, sentinel)`
// where the sentinel is a package-qualified value (FieldAccess like `sql.ErrNoRows`,
// `io.EOF`, etc.). These should use `errors.Is` for proper wrapped error support.
type PreferErrorsIsForFieldAccess struct {
	recipe.Base
}

func (r *PreferErrorsIsForFieldAccess) Name() string {
	return "org.openrewrite.golang.codequality.PreferErrorsIsForFieldAccess"
}
func (r *PreferErrorsIsForFieldAccess) DisplayName() string {
	return "Prefer errors.Is for package-qualified sentinel comparison"
}
func (r *PreferErrorsIsForFieldAccess) Description() string {
	return "Replace `err == sentinel` with `errors.Is(err, sentinel)` where the sentinel is a package-qualified value (e.g., `sql.ErrNoRows`). Use `errors.Is` for correct wrapped error handling."
}
func (r *PreferErrorsIsForFieldAccess) Tags() []string { return []string{"error-handling", "lint"} }

func (r *PreferErrorsIsForFieldAccess) Editor() recipe.TreeVisitor {
	return visitor.Init(&preferErrorsIsForFieldAccessVisitor{})
}

type preferErrorsIsForFieldAccessVisitor struct {
	visitor.GoVisitor
}

func (v *preferErrorsIsForFieldAccessVisitor) VisitBinary(bin *tree.Binary, p any) tree.J {
	bin = v.GoVisitor.VisitBinary(bin, p).(*tree.Binary)

	if bin.Operator.Element != tree.Equal && bin.Operator.Element != tree.NotEqual {
		return bin
	}

	leftIsFieldAccess := isPackageQualifiedSentinel(bin.Left)
	rightIsFieldAccess := isPackageQualifiedSentinel(bin.Right)

	if !leftIsFieldAccess && !rightIsFieldAccess {
		return bin
	}

	// Determine which side is the sentinel and which is the error expression.
	var errExpr, sentinel tree.Expression
	if rightIsFieldAccess {
		errExpr = bin.Left
		sentinel = bin.Right
	} else {
		errExpr = bin.Right
		sentinel = bin.Left
	}

	// Skip `err == nil` — nil is an Identifier, not a FieldAccess, but be safe.
	if ident, ok := sentinel.(*tree.Identifier); ok && ident.Name == "nil" {
		return bin
	}

	// Build errors.Is(errExpr, sentinel) or !errors.Is(errExpr, sentinel)
	prefix := getLeadingPrefixExpr(bin)

	errorsIdent := &tree.Identifier{Prefix: prefix, Name: "errors"}
	isIdent := &tree.Identifier{Name: "Is"}

	errArg := stripExprPrefix(errExpr)
	sentinelArg := stripExprPrefix(sentinel)
	sentinelArgWithSpace := setExprPrefixLocal(sentinelArg, tree.Space{Whitespace: " "})

	isCall := &tree.MethodInvocation{
		Select: &tree.RightPadded[tree.Expression]{Element: errorsIdent},
		Name:   isIdent,
		Arguments: tree.Container[tree.Expression]{
			Elements: []tree.RightPadded[tree.Expression]{
				{Element: errArg},
				{Element: sentinelArgWithSpace},
			},
		},
	}

	if bin.Operator.Element == tree.NotEqual {
		return &tree.Unary{
			Prefix:   prefix,
			Operator: tree.LeftPadded[tree.UnaryOperator]{Element: tree.Not},
			Operand:  setMethodInvocationPrefix(isCall, tree.Space{}),
		}
	}
	return isCall
}

// isPackageQualifiedSentinel checks if the expression is a FieldAccess
// whose field name starts with "Err" or is "EOF", indicating a package-qualified
// error sentinel (e.g., sql.ErrNoRows, io.EOF).
func isPackageQualifiedSentinel(expr tree.Expression) bool {
	fa, ok := expr.(*tree.FieldAccess)
	if !ok {
		return false
	}
	name := fa.Name.Element.Name
	if len(name) >= 3 && name[:3] == "Err" {
		return true
	}
	return name == "EOF"
}
