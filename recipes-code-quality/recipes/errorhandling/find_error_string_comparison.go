/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindErrorStringComparison finds `err.Error() == "some string"` comparisons.
// Comparing error messages by string is fragile. Use `errors.Is` or `errors.As` instead.
type FindErrorStringComparison struct {
	recipe.Base
}

func (r *FindErrorStringComparison) Name() string {
	return "org.openrewrite.golang.codequality.FindErrorStringComparison"
}
func (r *FindErrorStringComparison) DisplayName() string { return "Find error string comparison" }
func (r *FindErrorStringComparison) Description() string {
	return "Find `err.Error() == \"string\"` comparisons. Comparing error messages by string is fragile; use `errors.Is` or `errors.As` instead."
}
func (r *FindErrorStringComparison) Tags() []string { return []string{"error-handling", "lint"} }

func (r *FindErrorStringComparison) Editor() recipe.TreeVisitor {
	return visitor.Init(&findErrorStringComparisonVisitor{})
}

type findErrorStringComparisonVisitor struct {
	visitor.GoVisitor
}

func (v *findErrorStringComparisonVisitor) VisitBinary(bin *tree.Binary, p any) tree.J {
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
		tree.FoundSearchResult(bin.Markers, "comparing error string is fragile; use errors.Is or errors.As"),
	)
	return bin
}

// isErrorMethodCall checks if the expression is a method invocation with Name "Error"
// (e.g., err.Error()).
func isErrorMethodCall(expr tree.Expression) bool {
	mi, ok := expr.(*tree.MethodInvocation)
	if !ok {
		return false
	}
	return mi.Select != nil && mi.Name.Name == "Error"
}

// isStringLiteral checks if the expression is a string literal.
func isStringLiteral(expr tree.Expression) bool {
	lit, ok := expr.(*tree.Literal)
	if !ok {
		return false
	}
	// String literals have Source starting with " or `
	if len(lit.Source) < 2 {
		return false
	}
	return lit.Source[0] == '"' || lit.Source[0] == '`'
}
