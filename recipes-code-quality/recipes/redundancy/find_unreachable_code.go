/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package redundancy

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindUnreachableCode finds statements that appear after a return statement
// in the same block, which can never be executed.
type FindUnreachableCode struct {
	recipe.Base
}

func (r *FindUnreachableCode) Name() string {
	return "org.openrewrite.golang.codequality.FindUnreachableCode"
}
func (r *FindUnreachableCode) DisplayName() string { return "Find unreachable code" }
func (r *FindUnreachableCode) Description() string {
	return "Find statements after a `return` in the same block which are unreachable."
}
func (r *FindUnreachableCode) Tags() []string { return []string{"cleanup", "redundancy", "lint"} }

func (r *FindUnreachableCode) Editor() recipe.TreeVisitor {
	return visitor.Init(&findUnreachableCodeVisitor{})
}

type findUnreachableCodeVisitor struct {
	visitor.GoVisitor
}

func (v *findUnreachableCodeVisitor) VisitBlock(block *tree.Block, p any) tree.J {
	block = v.GoVisitor.VisitBlock(block, p).(*tree.Block)

	stmts := block.Statements
	if len(stmts) < 2 {
		return block
	}

	// Check if a return appears before the last statement, which means
	// there is unreachable code after it. We mark the return itself.
	for i := 0; i < len(stmts)-1; i++ {
		ret, ok := stmts[i].Element.(*tree.Return)
		if !ok {
			continue
		}

		// Found a return that is not the last statement -- code after it is unreachable.
		marked := ret.WithMarkers(
			tree.FoundSearchResult(ret.Markers, "unreachable code follows this return"),
		)
		stmts[i] = tree.RightPadded[tree.Statement]{
			Element: marked,
			After:   stmts[i].After,
			Markers: stmts[i].Markers,
		}
		block = block.WithStatements(stmts)
		return block
	}

	return block
}
