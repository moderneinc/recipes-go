/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindSqlPrepare finds calls to `db.Prepare()`. The returned *sql.Stmt must
// be closed when no longer needed to avoid leaking resources. Typically this
// is done with `defer stmt.Close()`.
type FindSqlPrepare struct {
	recipe.Base
}

func (r *FindSqlPrepare) Name() string {
	return "org.openrewrite.golang.codequality.FindSqlPrepare"
}
func (r *FindSqlPrepare) DisplayName() string { return "Find db.Prepare calls" }
func (r *FindSqlPrepare) Description() string {
	return "Find calls to `db.Prepare`. The returned prepared statement must be closed to avoid resource leaks."
}
func (r *FindSqlPrepare) Tags() []string { return []string{"style", "database/sql"} }

func (r *FindSqlPrepare) Editor() recipe.TreeVisitor {
	return visitor.Init(&findSqlPrepareVisitor{})
}

type findSqlPrepareVisitor struct {
	visitor.GoVisitor
}

func (v *findSqlPrepareVisitor) VisitMethodInvocation(mi *tree.MethodInvocation, p any) tree.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*tree.MethodInvocation)

	if mi.Select == nil {
		return mi
	}

	if mi.Name.Name != "Prepare" {
		return mi
	}

	mi = mi.WithMarkers(tree.FoundSearchResult(mi.Markers, "ensure prepared statement is closed"))
	return mi
}
