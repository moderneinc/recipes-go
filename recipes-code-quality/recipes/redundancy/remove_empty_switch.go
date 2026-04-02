/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package redundancy

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// RemoveEmptySwitch removes `switch` statements with no case clauses.
// An empty switch body is dead code that can be removed.
type RemoveEmptySwitch struct {
	recipe.Base
}

func (r *RemoveEmptySwitch) Name() string {
	return "org.openrewrite.golang.codequality.RemoveEmptySwitch"
}
func (r *RemoveEmptySwitch) DisplayName() string { return "Remove empty switch" }
func (r *RemoveEmptySwitch) Description() string {
	return "Remove `switch` statements with no case clauses. An empty switch body is dead code."
}
func (r *RemoveEmptySwitch) Tags() []string { return []string{"cleanup", "redundancy"} }

func (r *RemoveEmptySwitch) Editor() recipe.TreeVisitor {
	return visitor.Init(&removeEmptySwitchVisitor{})
}

type removeEmptySwitchVisitor struct {
	visitor.GoVisitor
}

func (v *removeEmptySwitchVisitor) VisitSwitch(sw *tree.Switch, p any) tree.J {
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

	// Remove the empty switch by replacing with Empty.
	return &tree.Empty{}
}
