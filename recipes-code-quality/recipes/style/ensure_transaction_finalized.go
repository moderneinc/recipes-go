/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// EnsureTransactionFinalized finds calls to `db.Begin()` and inserts
// `defer tx.Rollback()` after the assignment.
type EnsureTransactionFinalized struct {
	recipe.Base
}

func (r *EnsureTransactionFinalized) Name() string {
	return "org.openrewrite.golang.codequality.EnsureTransactionFinalized"
}
func (r *EnsureTransactionFinalized) DisplayName() string { return "Ensure transaction finalized" }
func (r *EnsureTransactionFinalized) Description() string {
	return "Find calls to `db.Begin`. Transactions must be committed or rolled back to avoid holding database locks."
}
func (r *EnsureTransactionFinalized) Tags() []string { return []string{"style", "database/sql"} }

func (r *EnsureTransactionFinalized) Editor() recipe.TreeVisitor {
	return visitor.Init(&ensureTransactionFinalizedVisitor{})
}

type ensureTransactionFinalizedVisitor struct {
	visitor.GoVisitor
}

func (v *ensureTransactionFinalizedVisitor) VisitBlock(block *java.Block, p any) java.J {
	block = v.GoVisitor.VisitBlock(block, p).(*java.Block)

	var newStmts []java.RightPadded[java.Statement]
	changed := false

	for i, rp := range block.Statements {
		newStmts = append(newStmts, rp)

		if varName, ok := extractAssignedVar(rp.Element, isDbBegin); ok {
			if hasDeferAfter(block.Statements, i, varName, "Rollback") {
				continue
			}
			deferStmt := buildDeferMethodCall(varName, "Rollback", rp.Element)
			newStmts = append(newStmts, java.RightPadded[java.Statement]{Element: deferStmt})
			changed = true
		}
	}

	if changed {
		return block.WithStatements(newStmts)
	}
	return block
}

// isDbBegin returns true if the method invocation is *.Begin().
func isDbBegin(mi *java.MethodInvocation) bool {
	if mi.Select == nil {
		return false
	}
	return mi.Name.Name == "Begin"
}
