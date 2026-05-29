/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package redundancy

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// RemoveUnreachableCode removes statements that appear after a return statement
// in the same block, which can never be executed.
type RemoveUnreachableCode struct {
	recipe.Base
}

func (r *RemoveUnreachableCode) Name() string {
	return "org.openrewrite.golang.codequality.RemoveUnreachableCode"
}
func (r *RemoveUnreachableCode) DisplayName() string { return "Remove unreachable code" }
func (r *RemoveUnreachableCode) Description() string {
	return "Remove statements after a `return` in the same block which are unreachable."
}
func (r *RemoveUnreachableCode) Tags() []string { return []string{"cleanup", "redundancy", "lint"} }

func (r *RemoveUnreachableCode) Editor() recipe.TreeVisitor {
	return visitor.Init(&removeUnreachableCodeVisitor{})
}

type removeUnreachableCodeVisitor struct {
	visitor.GoVisitor
}

func (v *removeUnreachableCodeVisitor) VisitBlock(block *java.Block, p any) java.J {
	block = v.GoVisitor.VisitBlock(block, p).(*java.Block)

	stmts := block.Statements
	if len(stmts) < 2 {
		return block
	}

	// Find the first return that is not the last statement and truncate
	// the statement list to remove unreachable code after it.
	for i := 0; i < len(stmts)-1; i++ {
		if _, ok := stmts[i].Element.(*java.Return); !ok {
			continue
		}

		// Found a return that is not the last statement -- remove everything after it.
		block = block.WithStatements(stmts[:i+1])
		return block
	}

	return block
}
