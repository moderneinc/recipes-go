/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindErrorTypeAssertion finds direct type assertions on errors like
// `err.(*MyError)`. These should use `errors.As` instead, which correctly
// handles wrapped errors.
type FindErrorTypeAssertion struct {
	recipe.Base
}

func (r *FindErrorTypeAssertion) Name() string {
	return "org.openrewrite.golang.codequality.FindErrorTypeAssertion"
}
func (r *FindErrorTypeAssertion) DisplayName() string {
	return "Find direct type assertion on error"
}
func (r *FindErrorTypeAssertion) Description() string {
	return "Find direct type assertions on errors like `err.(*MyError)`. Use `errors.As` instead for correct wrapped error handling."
}
func (r *FindErrorTypeAssertion) Tags() []string { return []string{"errorhandling", "lint"} }

func (r *FindErrorTypeAssertion) Editor() recipe.TreeVisitor {
	return visitor.Init(&findErrorTypeAssertionVisitor{})
}

type findErrorTypeAssertionVisitor struct {
	visitor.GoVisitor
}

func (v *findErrorTypeAssertionVisitor) VisitTypeCast(tc *tree.TypeCast, p any) tree.J {
	tc = v.GoVisitor.VisitTypeCast(tc, p).(*tree.TypeCast)

	// Check if the expression being asserted is an identifier named "err".
	ident, ok := tc.Expr.(*tree.Identifier)
	if !ok || ident.Name != "err" {
		return tc
	}

	tc = tc.WithMarkers(
		tree.FoundSearchResult(tc.Markers, "use errors.As instead of direct type assertion on error"),
	)
	return tc
}
