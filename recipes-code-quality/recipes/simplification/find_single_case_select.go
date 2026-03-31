/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindSingleCaseSelect finds `select` statements with exactly one
// communication clause and no default case. A single-case select without
// default is equivalent to the bare channel operation and the wrapping
// select is unnecessary.
type FindSingleCaseSelect struct {
	recipe.Base
}

func (r *FindSingleCaseSelect) Name() string {
	return "org.openrewrite.golang.codequality.FindSingleCaseSelect"
}
func (r *FindSingleCaseSelect) DisplayName() string { return "Find single-case select" }
func (r *FindSingleCaseSelect) Description() string {
	return "Find `select` statements with a single case and no default. The select is unnecessary and the channel operation can be used directly."
}
func (r *FindSingleCaseSelect) Tags() []string { return []string{"simplification", "cleanup"} }

func (r *FindSingleCaseSelect) Editor() recipe.TreeVisitor {
	return visitor.Init(&findSingleCaseSelectVisitor{})
}

type findSingleCaseSelectVisitor struct {
	visitor.GoVisitor
}

func (v *findSingleCaseSelectVisitor) VisitSwitch(sw *tree.Switch, p any) tree.J {
	sw = v.GoVisitor.VisitSwitch(sw, p).(*tree.Switch)

	// Only select statements (Switch with SelectStmt marker)
	if !tree.HasMarker[tree.SelectStmt](sw.Markers) {
		return sw
	}

	if sw.Body == nil {
		return sw
	}

	// Count CommClauses; a default CommClause has Comm == nil
	clauses := 0
	hasDefault := false
	for _, stmt := range sw.Body.Statements {
		if cc, ok := stmt.Element.(*tree.CommClause); ok {
			clauses++
			if cc.Comm == nil {
				hasDefault = true
			}
		}
	}

	// Flag only when there is exactly one comm clause and no default
	if clauses == 1 && !hasDefault {
		sw = sw.WithMarkers(
			tree.FoundSearchResult(sw.Markers, "single-case select without default is unnecessary; use the channel operation directly"),
		)
	}

	return sw
}
