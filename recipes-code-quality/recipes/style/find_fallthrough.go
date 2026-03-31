/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindFallthrough finds fallthrough statements in switch cases.
// Fallthrough is rarely used in Go and can be confusing to readers.
type FindFallthrough struct {
	recipe.Base
}

func (r *FindFallthrough) Name() string {
	return "org.openrewrite.golang.codequality.FindFallthrough"
}
func (r *FindFallthrough) DisplayName() string { return "Find fallthrough statements" }
func (r *FindFallthrough) Description() string {
	return "Find fallthrough statements in switch cases. Fallthrough is rarely used in Go and can be confusing."
}
func (r *FindFallthrough) Tags() []string { return []string{"style"} }

func (r *FindFallthrough) Editor() recipe.TreeVisitor {
	return visitor.Init(&findFallthroughVisitor{})
}

type findFallthroughVisitor struct {
	visitor.GoVisitor
}

func (v *findFallthroughVisitor) VisitFallthrough(f *tree.Fallthrough, p any) tree.J {
	f = v.GoVisitor.VisitFallthrough(f, p).(*tree.Fallthrough)
	f = f.WithMarkers(tree.FoundSearchResult(f.Markers, "consider removing fallthrough"))
	return f
}
