/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// AvoidLabel finds labeled statements. Labels are rarely needed and often
// indicate complex control flow that could be simplified.
type AvoidLabel struct {
	recipe.Base
}

func (r *AvoidLabel) Name() string {
	return "org.openrewrite.golang.codequality.AvoidLabel"
}
func (r *AvoidLabel) DisplayName() string { return "Avoid label" }
func (r *AvoidLabel) Description() string {
	return "Find labeled statements. Labels are rarely needed and indicate complex control flow."
}
func (r *AvoidLabel) Tags() []string { return []string{"style"} }

func (r *AvoidLabel) Editor() recipe.TreeVisitor {
	return visitor.Init(&avoidLabelVisitor{})
}

type avoidLabelVisitor struct {
	visitor.GoVisitor
}

func (v *avoidLabelVisitor) VisitLabel(l *tree.Label, p any) tree.J {
	l = v.GoVisitor.VisitLabel(l, p).(*tree.Label)
	l = l.WithMarkers(tree.MarkupInfo(l.Markers, "labeled statement indicates complex control flow"))
	return l
}
