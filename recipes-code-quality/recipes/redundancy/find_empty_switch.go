/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package redundancy

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindEmptySwitch finds `switch` statements with no case clauses.
// An empty switch body is dead code that can be removed.
type FindEmptySwitch struct {
	recipe.Base
}

func (r *FindEmptySwitch) Name() string {
	return "org.openrewrite.golang.codequality.FindEmptySwitch"
}
func (r *FindEmptySwitch) DisplayName() string { return "Find empty switch" }
func (r *FindEmptySwitch) Description() string {
	return "Find `switch` statements with no case clauses. An empty switch body is dead code that can be removed."
}
func (r *FindEmptySwitch) Tags() []string { return []string{"cleanup", "redundancy"} }

func (r *FindEmptySwitch) Editor() recipe.TreeVisitor {
	return visitor.Init(&findEmptySwitchVisitor{})
}

type findEmptySwitchVisitor struct {
	visitor.GoVisitor
}

func (v *findEmptySwitchVisitor) VisitSwitch(sw *tree.Switch, p any) tree.J {
	sw = v.GoVisitor.VisitSwitch(sw, p).(*tree.Switch)

	// Skip select statements.
	if tree.HasMarker[tree.SelectStmt](sw.Markers) {
		return sw
	}

	// Body must exist but contain no real statements.
	if sw.Body == nil {
		return sw
	}
	for _, stmt := range sw.Body.Statements {
		if _, isEmpty := stmt.Element.(*tree.Empty); !isEmpty {
			return sw
		}
	}

	sw = sw.WithMarkers(
		tree.FoundSearchResult(sw.Markers, "empty switch statement"),
	)
	return sw
}
