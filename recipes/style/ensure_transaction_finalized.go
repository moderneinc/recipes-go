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
	return insertDeferMethodCall(block, isSqlTx, "Rollback")
}

func isSqlTx(a acquisition) bool {
	return typeIs(a.varType, "database/sql.Tx")
}
