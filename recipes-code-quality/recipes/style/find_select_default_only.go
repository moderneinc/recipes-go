/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindSelectDefaultOnly finds `select { default: ... }` statements where the
// select has only a default case and no communication cases. Such a select is
// unnecessary and the body can be inlined directly.
type FindSelectDefaultOnly struct {
	recipe.Base
}

func (r *FindSelectDefaultOnly) Name() string {
	return "org.openrewrite.golang.codequality.FindSelectDefaultOnly"
}
func (r *FindSelectDefaultOnly) DisplayName() string { return "Find select with only default case" }
func (r *FindSelectDefaultOnly) Description() string {
	return "Find `select { default: ... }` statements with only a default case and no communication cases. The select is unnecessary and the body can be inlined."
}
func (r *FindSelectDefaultOnly) Tags() []string { return []string{"style"} }

func (r *FindSelectDefaultOnly) Editor() recipe.TreeVisitor {
	return visitor.Init(&findSelectDefaultOnlyVisitor{})
}

type findSelectDefaultOnlyVisitor struct {
	visitor.GoVisitor
}

func (v *findSelectDefaultOnlyVisitor) VisitSwitch(sw *tree.Switch, p any) tree.J {
	sw = v.GoVisitor.VisitSwitch(sw, p).(*tree.Switch)

	// Only select statements (Switch with SelectStmt marker)
	if !tree.HasMarker[tree.SelectStmt](sw.Markers) {
		return sw
	}

	if sw.Body == nil {
		return sw
	}

	// Count CommClauses and check if the only one is a default (Comm == nil)
	clauses := 0
	allDefault := true
	for _, stmt := range sw.Body.Statements {
		cc, ok := stmt.Element.(*tree.CommClause)
		if !ok {
			continue
		}
		clauses++
		if cc.Comm != nil {
			allDefault = false
		}
	}

	// Flag only when there is exactly one clause and it is a default
	if clauses == 1 && allDefault {
		sw = sw.WithMarkers(
			tree.FoundSearchResult(sw.Markers, "select with only a default case is unnecessary; inline the body"),
		)
	}

	return sw
}
