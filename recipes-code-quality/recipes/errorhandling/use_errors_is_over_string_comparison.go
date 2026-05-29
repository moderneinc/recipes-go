/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
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

func (v *useErrorsIsOverStringComparisonVisitor) VisitBinary(bin *java.Binary, p any) java.J {
	bin = v.GoVisitor.VisitBinary(bin, p).(*java.Binary)

	if bin.Operator.Element != java.Equal && bin.Operator.Element != java.NotEqual {
		return bin
	}

	// Check if one side is err.Error() and the other is a string literal.
	leftIsErrorCall := isErrorMethodCall(bin.Left)
	rightIsErrorCall := isErrorMethodCall(bin.Right)

	if !leftIsErrorCall && !rightIsErrorCall {
		return bin
	}

	var other java.Expression
	if leftIsErrorCall {
		other = bin.Right
	} else {
		other = bin.Left
	}

	if !isStringLiteral(other) {
		return bin
	}

	bin = bin.WithMarkers(
		java.MarkupWarn(bin.Markers, "comparing error string is fragile; use errors.Is or errors.As"),
	)
	return bin
}

// isErrorMethodCall checks if an expression is a method call of the form x.Error().
func isErrorMethodCall(expr java.Expression) bool {
	mi, ok := expr.(*java.MethodInvocation)
	if !ok {
		return false
	}
	if mi.Select == nil {
		return false
	}
	return mi.Name.Name == "Error" && len(mi.Arguments.Elements) == 0
}

// isStringLiteral checks if an expression is a string Literal.
func isStringLiteral(expr java.Expression) bool {
	lit, ok := expr.(*java.Literal)
	return ok && lit.Kind == java.StringLiteral
}
