/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// EnsureSqlRowsClosed finds calls to `db.Query()` and inserts
// `defer rows.Close()` after the assignment.
type EnsureSqlRowsClosed struct {
	recipe.Base
}

func (r *EnsureSqlRowsClosed) Name() string {
	return "org.openrewrite.golang.codequality.EnsureSqlRowsClosed"
}
func (r *EnsureSqlRowsClosed) DisplayName() string { return "Ensure SQL rows closed" }
func (r *EnsureSqlRowsClosed) Description() string {
	return "Find calls to `db.Query`. The returned rows must be closed with `defer rows.Close()` to avoid connection leaks."
}
func (r *EnsureSqlRowsClosed) Tags() []string { return []string{"style", "database/sql"} }

func (r *EnsureSqlRowsClosed) Editor() recipe.TreeVisitor {
	return visitor.Init(&ensureSqlRowsClosedVisitor{})
}

type ensureSqlRowsClosedVisitor struct {
	visitor.GoVisitor
}

func (v *ensureSqlRowsClosedVisitor) VisitBlock(block *tree.Block, p any) tree.J {
	block = v.GoVisitor.VisitBlock(block, p).(*tree.Block)

	var newStmts []tree.RightPadded[tree.Statement]
	changed := false

	for i, rp := range block.Statements {
		newStmts = append(newStmts, rp)

		if varName, ok := extractAssignedVar(rp.Element, isSqlQuery); ok {
			if hasDeferAfter(block.Statements, i, varName, "Close") {
				continue
			}
			deferStmt := buildDeferMethodCall(varName, "Close", rp.Element)
			newStmts = append(newStmts, tree.RightPadded[tree.Statement]{Element: deferStmt})
			changed = true
		}
	}

	if changed {
		return block.WithStatements(newStmts)
	}
	return block
}

// isSqlQuery returns true if the method invocation is *.Query().
func isSqlQuery(mi *tree.MethodInvocation) bool {
	if mi.Select == nil {
		return false
	}
	return mi.Name.Name == "Query"
}
