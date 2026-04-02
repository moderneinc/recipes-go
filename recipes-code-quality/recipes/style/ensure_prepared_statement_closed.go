/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree"
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

func (r *EnsurePreparedStatementClosed) Editor() recipe.TreeVisitor {
	return visitor.Init(&ensurePreparedStatementClosedVisitor{})
}

type ensurePreparedStatementClosedVisitor struct {
	visitor.GoVisitor
}

func (v *ensurePreparedStatementClosedVisitor) VisitBlock(block *tree.Block, p any) tree.J {
	block = v.GoVisitor.VisitBlock(block, p).(*tree.Block)

	var newStmts []tree.RightPadded[tree.Statement]
	changed := false

	for i, rp := range block.Statements {
		newStmts = append(newStmts, rp)

		if varName, ok := extractAssignedVar(rp.Element, isDbPrepare); ok {
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

// isDbPrepare returns true if the method invocation is *.Prepare().
func isDbPrepare(mi *tree.MethodInvocation) bool {
	if mi.Select == nil {
		return false
	}
	return mi.Name.Name == "Prepare"
}
