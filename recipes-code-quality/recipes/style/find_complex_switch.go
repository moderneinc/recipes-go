/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindComplexSwitch finds switch statements with more than 10 cases.
// A switch with many cases suggests using a map or strategy pattern instead.
type FindComplexSwitch struct {
	recipe.Base
}

func (r *FindComplexSwitch) Name() string {
	return "org.openrewrite.golang.codequality.FindComplexSwitch"
}
func (r *FindComplexSwitch) DisplayName() string { return "Find complex switch statements" }
func (r *FindComplexSwitch) Description() string {
	return "Find switch statements with more than 10 cases. Consider using a map or strategy pattern."
}
func (r *FindComplexSwitch) Tags() []string { return []string{"style"} }

func (r *FindComplexSwitch) Editor() recipe.TreeVisitor {
	return visitor.Init(&findComplexSwitchVisitor{})
}

type findComplexSwitchVisitor struct {
	visitor.GoVisitor
}

func (v *findComplexSwitchVisitor) VisitSwitch(sw *tree.Switch, p any) tree.J {
	sw = v.GoVisitor.VisitSwitch(sw, p).(*tree.Switch)

	if sw.Body == nil {
		return sw
	}

	count := 0
	for _, stmt := range sw.Body.Statements {
		if _, ok := stmt.Element.(*tree.Case); ok {
			count++
		}
	}

	if count <= 10 {
		return sw
	}

	sw = sw.WithMarkers(tree.FoundSearchResult(sw.Markers, "switch has too many cases; consider using a map or strategy pattern"))
	return sw
}
