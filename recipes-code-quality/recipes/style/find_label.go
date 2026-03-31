/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindLabel finds labeled statements. Labels are rarely needed and often
// indicate complex control flow that could be simplified.
type FindLabel struct {
	recipe.Base
}

func (r *FindLabel) Name() string {
	return "org.openrewrite.golang.codequality.FindLabel"
}
func (r *FindLabel) DisplayName() string { return "Find labeled statements" }
func (r *FindLabel) Description() string {
	return "Find labeled statements. Labels are rarely needed and indicate complex control flow."
}
func (r *FindLabel) Tags() []string { return []string{"style"} }

func (r *FindLabel) Editor() recipe.TreeVisitor {
	return visitor.Init(&findLabelVisitor{})
}

type findLabelVisitor struct {
	visitor.GoVisitor
}

func (v *findLabelVisitor) VisitLabel(l *tree.Label, p any) tree.J {
	l = v.GoVisitor.VisitLabel(l, p).(*tree.Label)
	l = l.WithMarkers(tree.FoundSearchResult(l.Markers, "labeled statement indicates complex control flow"))
	return l
}
