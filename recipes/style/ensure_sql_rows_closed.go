/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/moderneinc/recipes-go/diagnostic"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// EnsureSqlRowsClosed finds assignments of a `*sql.Rows` and inserts
// `defer rows.Close()` after the assignment.
type EnsureSqlRowsClosed struct {
	recipe.Base
}

func (r *EnsureSqlRowsClosed) Name() string {
	return "org.openrewrite.golang.codequality.EnsureSqlRowsClosed"
}
func (r *EnsureSqlRowsClosed) DisplayName() string { return "Ensure SQL rows closed" }
func (r *EnsureSqlRowsClosed) Description() string {
	return "Find assignments of a `*sql.Rows`, as returned by `db.Query`. The rows must be closed with `defer rows.Close()` to avoid connection leaks."
}
func (r *EnsureSqlRowsClosed) Tags() []string { return []string{"style", "database/sql"} }

func (r *EnsureSqlRowsClosed) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{
		{DiagnosticID: "sqlclosecheck", Tool: diagnostic.GolangciLint, HasFix: false},
	}
}

func (r *EnsureSqlRowsClosed) Editor() recipe.TreeVisitor {
	return visitor.Init(&ensureSqlRowsClosedVisitor{})
}

type ensureSqlRowsClosedVisitor struct {
	visitor.GoVisitor
}

func (v *ensureSqlRowsClosedVisitor) VisitBlock(block *java.Block, p any) java.J {
	block = v.GoVisitor.VisitBlock(block, p).(*java.Block)
	return insertDeferMethodCall(block, isSqlRows, "Close")
}

func isSqlRows(a acquisition) bool {
	return typeIs(a.varType, "database/sql.Rows")
}
