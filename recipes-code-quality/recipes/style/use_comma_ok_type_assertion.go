/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// UseCommaOkTypeAssertion finds type assertions `x.(T)`. Bare type
// assertions panic on failure; consider using the comma-ok form
// `v, ok := x.(T)` instead.
type UseCommaOkTypeAssertion struct {
	recipe.Base
}

func (r *UseCommaOkTypeAssertion) Name() string {
	return "org.openrewrite.golang.codequality.UseCommaOkTypeAssertion"
}
func (r *UseCommaOkTypeAssertion) DisplayName() string {
	return "Use comma-ok type assertion"
}
func (r *UseCommaOkTypeAssertion) Description() string {
	return "Find type assertions `x.(T)`. Bare type assertions panic on failure; consider using the comma-ok form."
}
func (r *UseCommaOkTypeAssertion) Tags() []string { return []string{"style", "lint"} }

func (r *UseCommaOkTypeAssertion) Editor() recipe.TreeVisitor {
	return visitor.Init(&useCommaOkTypeAssertionVisitor{})
}

type useCommaOkTypeAssertionVisitor struct {
	visitor.GoVisitor
}

func (v *useCommaOkTypeAssertionVisitor) VisitTypeCast(tc *tree.TypeCast, p any) tree.J {
	tc = v.GoVisitor.VisitTypeCast(tc, p).(*tree.TypeCast)

	tc = tc.WithMarkers(tree.MarkupWarn(tc.Markers, "consider using comma-ok form for type assertion"))
	return tc
}
