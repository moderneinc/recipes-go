/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindPanic finds calls to the built-in `panic()` function. Panics crash the
// entire program and should generally be avoided in library code and goroutines.
type FindPanic struct {
	recipe.Base
}

func (r *FindPanic) Name() string {
	return "org.openrewrite.golang.codequality.FindPanic"
}
func (r *FindPanic) DisplayName() string { return "Find panic calls" }
func (r *FindPanic) Description() string {
	return "Find calls to the built-in `panic()` function, which crashes the program."
}
func (r *FindPanic) Tags() []string { return []string{"errorhandling", "lint"} }

func (r *FindPanic) Editor() recipe.TreeVisitor {
	return visitor.Init(&findPanicVisitor{})
}

type findPanicVisitor struct {
	visitor.GoVisitor
}

func (v *findPanicVisitor) VisitMethodInvocation(mi *tree.MethodInvocation, p any) tree.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*tree.MethodInvocation)

	// Match: panic(...) — built-in, so no Select and Name == "panic".
	if mi.Select != nil || mi.Name.Name != "panic" {
		return mi
	}

	mi = mi.WithMarkers(
		tree.FoundSearchResult(mi.Markers, "panic call found; consider returning an error instead"),
	)
	return mi
}
