/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindSqlBegin finds calls to `db.Begin()`. Transactions must be committed
// or rolled back to avoid holding database locks and leaking connections.
// Typically a deferred rollback is used as a safety net alongside an explicit
// commit on the success path.
type FindSqlBegin struct {
	recipe.Base
}

func (r *FindSqlBegin) Name() string {
	return "org.openrewrite.golang.codequality.FindSqlBegin"
}
func (r *FindSqlBegin) DisplayName() string { return "Find db.Begin calls" }
func (r *FindSqlBegin) Description() string {
	return "Find calls to `db.Begin`. Transactions must be committed or rolled back to avoid holding database locks."
}
func (r *FindSqlBegin) Tags() []string { return []string{"style", "database/sql"} }

func (r *FindSqlBegin) Editor() recipe.TreeVisitor {
	return visitor.Init(&findSqlBeginVisitor{})
}

type findSqlBeginVisitor struct {
	visitor.GoVisitor
}

func (v *findSqlBeginVisitor) VisitMethodInvocation(mi *tree.MethodInvocation, p any) tree.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*tree.MethodInvocation)

	if mi.Select == nil {
		return mi
	}

	if mi.Name.Name != "Begin" {
		return mi
	}

	mi = mi.WithMarkers(tree.FoundSearchResult(mi.Markers, "ensure transaction is committed or rolled back"))
	return mi
}
