/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// AvoidGoto finds goto statements. Goto makes control flow hard to follow
// and should generally be restructured using loops, conditionals, or functions.
type AvoidGoto struct {
	recipe.Base
}

func (r *AvoidGoto) Name() string {
	return "org.openrewrite.golang.codequality.AvoidGoto"
}
func (r *AvoidGoto) DisplayName() string { return "Avoid goto" }
func (r *AvoidGoto) Description() string {
	return "Find goto statements. Goto makes control flow hard to follow and should be restructured."
}
func (r *AvoidGoto) Tags() []string { return []string{"style"} }

func (r *AvoidGoto) Editor() recipe.TreeVisitor {
	return visitor.Init(&avoidGotoVisitor{})
}

type avoidGotoVisitor struct {
	visitor.GoVisitor
}

func (v *avoidGotoVisitor) VisitGoto(g *tree.Goto, p any) tree.J {
	g = v.GoVisitor.VisitGoto(g, p).(*tree.Goto)
	g = g.WithMarkers(tree.MarkupWarn(g.Markers, "consider restructuring to avoid goto"))
	return g
}
