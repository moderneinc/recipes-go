/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindContextErr finds `ctx.Err()` calls. The context error should be inspected
// to distinguish between cancellation and deadline exceeded.
type FindContextErr struct {
	recipe.Base
}

func (r *FindContextErr) Name() string {
	return "org.openrewrite.golang.codequality.FindContextErr"
}
func (r *FindContextErr) DisplayName() string { return "Find ctx.Err() calls" }
func (r *FindContextErr) Description() string {
	return "Find `ctx.Err()` calls. The context error should be inspected to distinguish between cancellation and deadline exceeded."
}
func (r *FindContextErr) Tags() []string { return []string{"error-handling", "lint"} }

func (r *FindContextErr) Editor() recipe.TreeVisitor {
	return visitor.Init(&findContextErrVisitor{})
}

type findContextErrVisitor struct {
	visitor.GoVisitor
}

func (v *findContextErrVisitor) VisitMethodInvocation(mi *tree.MethodInvocation, p any) tree.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*tree.MethodInvocation)

	if mi.Select == nil {
		return mi
	}

	ident, ok := mi.Select.Element.(*tree.Identifier)
	if !ok || ident.Name != "ctx" {
		return mi
	}

	if mi.Name.Name != "Err" {
		return mi
	}

	mi = mi.WithMarkers(
		tree.FoundSearchResult(mi.Markers, "ctx.Err() found; inspect the context error"),
	)
	return mi
}
