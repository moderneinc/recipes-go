/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindTypeAssertionWithoutOk finds type assertions `x.(T)`. Bare type
// assertions panic on failure; consider using the comma-ok form
// `v, ok := x.(T)` instead.
type FindTypeAssertionWithoutOk struct {
	recipe.Base
}

func (r *FindTypeAssertionWithoutOk) Name() string {
	return "org.openrewrite.golang.codequality.FindTypeAssertionWithoutOk"
}
func (r *FindTypeAssertionWithoutOk) DisplayName() string {
	return "Find type assertions without ok check"
}
func (r *FindTypeAssertionWithoutOk) Description() string {
	return "Find type assertions `x.(T)`. Bare type assertions panic on failure; consider using the comma-ok form."
}
func (r *FindTypeAssertionWithoutOk) Tags() []string { return []string{"style", "lint"} }

func (r *FindTypeAssertionWithoutOk) Editor() recipe.TreeVisitor {
	return visitor.Init(&findTypeAssertionWithoutOkVisitor{})
}

type findTypeAssertionWithoutOkVisitor struct {
	visitor.GoVisitor
}

func (v *findTypeAssertionWithoutOkVisitor) VisitTypeCast(tc *tree.TypeCast, p any) tree.J {
	tc = v.GoVisitor.VisitTypeCast(tc, p).(*tree.TypeCast)

	tc = tc.WithMarkers(tree.FoundSearchResult(tc.Markers, "consider using comma-ok form for type assertion"))
	return tc
}
