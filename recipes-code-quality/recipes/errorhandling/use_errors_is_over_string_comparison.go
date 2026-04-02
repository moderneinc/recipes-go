/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// UseErrorsIsOverStringComparison finds `err.Error() == "some string"` comparisons.
// Comparing error messages by string is fragile. Use `errors.Is` or `errors.As` instead.
type UseErrorsIsOverStringComparison struct {
	recipe.Base
}

func (r *UseErrorsIsOverStringComparison) Name() string {
	return "org.openrewrite.golang.codequality.UseErrorsIsOverStringComparison"
}
func (r *UseErrorsIsOverStringComparison) DisplayName() string {
	return "Use errors.Is over string comparison"
}
func (r *UseErrorsIsOverStringComparison) Description() string {
	return "Find `err.Error() == \"string\"` comparisons. Comparing error messages by string is fragile; use `errors.Is` or `errors.As` instead."
}
func (r *UseErrorsIsOverStringComparison) Tags() []string {
	return []string{"error-handling", "lint"}
}

func (r *UseErrorsIsOverStringComparison) Editor() recipe.TreeVisitor {
	return visitor.Init(&useErrorsIsOverStringComparisonVisitor{})
}

type useErrorsIsOverStringComparisonVisitor struct {
	visitor.GoVisitor
}

func (v *useErrorsIsOverStringComparisonVisitor) VisitBinary(bin *tree.Binary, p any) tree.J {
	bin = v.GoVisitor.VisitBinary(bin, p).(*tree.Binary)

	if bin.Operator.Element != tree.Equal && bin.Operator.Element != tree.NotEqual {
		return bin
	}

	// Check if one side is err.Error() and the other is a string literal.
	leftIsErrorCall := isErrorMethodCall(bin.Left)
	rightIsErrorCall := isErrorMethodCall(bin.Right)

	if !leftIsErrorCall && !rightIsErrorCall {
		return bin
	}

	var other tree.Expression
	if leftIsErrorCall {
		other = bin.Right
	} else {
		other = bin.Left
	}

	if !isStringLiteral(other) {
		return bin
	}

	bin = bin.WithMarkers(
		tree.MarkupWarn(bin.Markers, "comparing error string is fragile; use errors.Is or errors.As"),
	)
	return bin
}

// isErrorMethodCall checks if an expression is a method call of the form x.Error().
func isErrorMethodCall(expr tree.Expression) bool {
	mi, ok := expr.(*tree.MethodInvocation)
	if !ok {
		return false
	}
	if mi.Select == nil {
		return false
	}
	return mi.Name.Name == "Error" && len(mi.Arguments.Elements) == 0
}

// isStringLiteral checks if an expression is a string Literal.
func isStringLiteral(expr tree.Expression) bool {
	lit, ok := expr.(*tree.Literal)
	return ok && lit.Kind == tree.StringLiteral
}
