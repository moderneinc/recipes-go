/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindRecover finds calls to the built-in `recover()` function. Recover
// catches panics and should only appear inside deferred functions. Its
// presence often signals unusual control flow that deserves review.
type FindRecover struct {
	recipe.Base
}

func (r *FindRecover) Name() string {
	return "org.openrewrite.golang.codequality.FindRecover"
}
func (r *FindRecover) DisplayName() string { return "Find recover() calls" }
func (r *FindRecover) Description() string {
	return "Find calls to the built-in `recover()` function, which catches panics and signals unusual control flow."
}
func (r *FindRecover) Tags() []string { return []string{"error-handling"} }

func (r *FindRecover) Editor() recipe.TreeVisitor {
	return visitor.Init(&findRecoverVisitor{})
}

type findRecoverVisitor struct {
	visitor.GoVisitor
}

func (v *findRecoverVisitor) VisitMethodInvocation(mi *tree.MethodInvocation, p any) tree.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*tree.MethodInvocation)

	// Match: recover() — built-in, so no Select and Name == "recover".
	if mi.Select != nil || mi.Name.Name != "recover" {
		return mi
	}

	mi = mi.WithMarkers(
		tree.FoundSearchResult(mi.Markers, "recover() catches panics; ensure it is in a deferred function"),
	)
	return mi
}
