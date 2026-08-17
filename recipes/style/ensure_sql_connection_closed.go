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

// EnsureSqlConnectionClosed finds calls to `sql.Open()` and inserts
// `defer db.Close()` after the assignment.
type EnsureSqlConnectionClosed struct {
	recipe.Base
}

func (r *EnsureSqlConnectionClosed) Name() string {
	return "org.openrewrite.golang.codequality.EnsureSqlConnectionClosed"
}
func (r *EnsureSqlConnectionClosed) DisplayName() string { return "Ensure SQL connection closed" }
func (r *EnsureSqlConnectionClosed) Description() string {
	return "Find calls to `sql.Open`. Database connections should be managed carefully and closed when no longer needed."
}
func (r *EnsureSqlConnectionClosed) Tags() []string { return []string{"style", "database/sql"} }

func (r *EnsureSqlConnectionClosed) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{
		{DiagnosticID: "sqlclosecheck", Tool: diagnostic.GolangciLint, HasFix: false},
	}
}

func (r *EnsureSqlConnectionClosed) Editor() recipe.TreeVisitor {
	return visitor.Init(&ensureSqlConnectionClosedVisitor{})
}

type ensureSqlConnectionClosedVisitor struct {
	visitor.GoVisitor
}

func (v *ensureSqlConnectionClosedVisitor) VisitBlock(block *java.Block, p any) java.J {
	block = v.GoVisitor.VisitBlock(block, p).(*java.Block)
	return insertDeferMethodCall(block, isSqlOpen, "Close")
}

// isSqlOpen returns true if the method invocation is sql.Open.
func isSqlOpen(a acquisition) bool {
	declaring, ok := declaringType(a.call)
	return ok && declaring == "database/sql" && a.call.Name.Name == "Open"
}
