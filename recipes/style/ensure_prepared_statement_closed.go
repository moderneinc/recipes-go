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

// EnsurePreparedStatementClosed finds calls to `db.Prepare()` and inserts
// `defer stmt.Close()` after the assignment.
type EnsurePreparedStatementClosed struct {
	recipe.Base
}

func (r *EnsurePreparedStatementClosed) Name() string {
	return "org.openrewrite.golang.codequality.EnsurePreparedStatementClosed"
}
func (r *EnsurePreparedStatementClosed) DisplayName() string {
	return "Ensure prepared statement closed"
}
func (r *EnsurePreparedStatementClosed) Description() string {
	return "Find calls to `db.Prepare`. The returned prepared statement must be closed to avoid resource leaks."
}
func (r *EnsurePreparedStatementClosed) Tags() []string { return []string{"style", "database/sql"} }

func (r *EnsurePreparedStatementClosed) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{
		{DiagnosticID: "sqlclosecheck", Tool: diagnostic.GolangciLint, HasFix: false},
	}
}

func (r *EnsurePreparedStatementClosed) Editor() recipe.TreeVisitor {
	return visitor.Init(&ensurePreparedStatementClosedVisitor{})
}

type ensurePreparedStatementClosedVisitor struct {
	visitor.GoVisitor
}

func (v *ensurePreparedStatementClosedVisitor) VisitBlock(block *java.Block, p any) java.J {
	block = v.GoVisitor.VisitBlock(block, p).(*java.Block)
	return insertDeferMethodCall(block, isSqlStmt, "Close")
}

func isSqlStmt(a acquisition) bool {
	return typeIs(a.varType, "database/sql.Stmt")
}
