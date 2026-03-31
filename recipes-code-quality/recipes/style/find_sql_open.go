/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindSqlOpen finds calls to `sql.Open()`. Database connections should be
// managed carefully — ensure the returned *sql.DB is closed when no longer
// needed and that connection pool settings are configured appropriately.
type FindSqlOpen struct {
	recipe.Base
}

func (r *FindSqlOpen) Name() string {
	return "org.openrewrite.golang.codequality.FindSqlOpen"
}
func (r *FindSqlOpen) DisplayName() string { return "Find sql.Open calls" }
func (r *FindSqlOpen) Description() string {
	return "Find calls to `sql.Open`. Database connections should be managed carefully and closed when no longer needed."
}
func (r *FindSqlOpen) Tags() []string { return []string{"style", "database/sql"} }

func (r *FindSqlOpen) Editor() recipe.TreeVisitor {
	return visitor.Init(&findSqlOpenVisitor{})
}

type findSqlOpenVisitor struct {
	visitor.GoVisitor
}

func (v *findSqlOpenVisitor) VisitMethodInvocation(mi *tree.MethodInvocation, p any) tree.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*tree.MethodInvocation)

	if mi.Select == nil {
		return mi
	}

	ident, ok := mi.Select.Element.(*tree.Identifier)
	if !ok || ident.Name != "sql" {
		return mi
	}

	if mi.Name.Name != "Open" {
		return mi
	}

	mi = mi.WithMarkers(tree.FoundSearchResult(mi.Markers, "ensure the database connection is closed"))
	return mi
}
