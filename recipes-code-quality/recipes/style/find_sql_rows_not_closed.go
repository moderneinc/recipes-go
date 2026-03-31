/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindSqlQuery finds calls to `db.Query()`. The returned *sql.Rows must be
// closed to avoid leaking database connections. Typically this is done with
// `defer rows.Close()` immediately after checking the error.
type FindSqlQuery struct {
	recipe.Base
}

func (r *FindSqlQuery) Name() string {
	return "org.openrewrite.golang.codequality.FindSqlQuery"
}
func (r *FindSqlQuery) DisplayName() string { return "Find db.Query calls" }
func (r *FindSqlQuery) Description() string {
	return "Find calls to `db.Query`. The returned rows must be closed with `defer rows.Close()` to avoid connection leaks."
}
func (r *FindSqlQuery) Tags() []string { return []string{"style", "database/sql"} }

func (r *FindSqlQuery) Editor() recipe.TreeVisitor {
	return visitor.Init(&findSqlQueryVisitor{})
}

type findSqlQueryVisitor struct {
	visitor.GoVisitor
}

func (v *findSqlQueryVisitor) VisitMethodInvocation(mi *tree.MethodInvocation, p any) tree.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*tree.MethodInvocation)

	if mi.Select == nil {
		return mi
	}

	if mi.Name.Name != "Query" {
		return mi
	}

	mi = mi.WithMarkers(tree.FoundSearchResult(mi.Markers, "ensure rows are closed with defer rows.Close()"))
	return mi
}
