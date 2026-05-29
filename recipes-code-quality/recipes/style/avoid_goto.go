/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
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

func (v *avoidGotoVisitor) VisitGoto(g *golang.Goto, p any) java.J {
	g = v.GoVisitor.VisitGoto(g, p).(*golang.Goto)
	g = g.WithMarkers(java.MarkupWarn(g.Markers, "consider restructuring to avoid goto"))
	return g
}
