/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
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

func (r *EnsureSqlConnectionClosed) Editor() recipe.TreeVisitor {
	return visitor.Init(&ensureSqlConnectionClosedVisitor{})
}

type ensureSqlConnectionClosedVisitor struct {
	visitor.GoVisitor
}

func (v *ensureSqlConnectionClosedVisitor) VisitBlock(block *java.Block, p any) java.J {
	block = v.GoVisitor.VisitBlock(block, p).(*java.Block)

	var newStmts []java.RightPadded[java.Statement]
	changed := false

	for i, rp := range block.Statements {
		newStmts = append(newStmts, rp)

		if varName, ok := extractAssignedVar(rp.Element, isSqlOpen); ok {
			if hasDeferAfter(block.Statements, i, varName, "Close") {
				continue
			}
			deferStmt := buildDeferMethodCall(varName, "Close", rp.Element)
			newStmts = append(newStmts, java.RightPadded[java.Statement]{Element: deferStmt})
			changed = true
		}
	}

	if changed {
		return block.WithStatements(newStmts)
	}
	return block
}

// isSqlOpen returns true if the method invocation is sql.Open.
func isSqlOpen(mi *java.MethodInvocation) bool {
	if mi.Select == nil {
		return false
	}
	ident, ok := mi.Select.Element.(*java.Identifier)
	if !ok || ident.Name != "sql" {
		return false
	}
	return mi.Name.Name == "Open"
}
