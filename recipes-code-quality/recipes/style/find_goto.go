/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindGoto finds goto statements. Goto makes control flow hard to follow
// and should generally be restructured using loops, conditionals, or functions.
type FindGoto struct {
	recipe.Base
}

func (r *FindGoto) Name() string {
	return "org.openrewrite.golang.codequality.FindGoto"
}
func (r *FindGoto) DisplayName() string { return "Find goto statements" }
func (r *FindGoto) Description() string {
	return "Find goto statements. Goto makes control flow hard to follow and should be restructured."
}
func (r *FindGoto) Tags() []string { return []string{"style"} }

func (r *FindGoto) Editor() recipe.TreeVisitor {
	return visitor.Init(&findGotoVisitor{})
}

type findGotoVisitor struct {
	visitor.GoVisitor
}

func (v *findGotoVisitor) VisitGoto(g *tree.Goto, p any) tree.J {
	g = v.GoVisitor.VisitGoto(g, p).(*tree.Goto)
	g = g.WithMarkers(tree.FoundSearchResult(g.Markers, "consider restructuring to avoid goto"))
	return g
}
