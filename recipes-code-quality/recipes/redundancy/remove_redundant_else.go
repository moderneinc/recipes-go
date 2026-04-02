/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package redundancy

import (
	"strings"

	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// RemoveRedundantElse removes `if cond { ... return } else { body }` patterns
// by lifting the else body out as sibling statements after the if, since the
// else is unreachable when the if block ends with a return.
type RemoveRedundantElse struct {
	recipe.Base
}

func (r *RemoveRedundantElse) Name() string {
	return "org.openrewrite.golang.codequality.RemoveRedundantElse"
}
func (r *RemoveRedundantElse) DisplayName() string { return "Remove redundant else after return" }
func (r *RemoveRedundantElse) Description() string {
	return "Remove `if ... { return } else { ... }` where the else is redundant because the if block ends with a return."
}
func (r *RemoveRedundantElse) Tags() []string { return []string{"cleanup", "redundancy"} }

func (r *RemoveRedundantElse) Editor() recipe.TreeVisitor {
	return visitor.Init(&removeRedundantElseVisitor{})
}

type removeRedundantElseVisitor struct {
	visitor.GoVisitor
}

func (v *removeRedundantElseVisitor) VisitBlock(block *tree.Block, p any) tree.J {
	block = v.GoVisitor.VisitBlock(block, p).(*tree.Block)

	var newStmts []tree.RightPadded[tree.Statement]
	changed := false

	for _, rp := range block.Statements {
		ifStmt, ok := rp.Element.(*tree.If)
		if !ok || ifStmt.ElsePart == nil {
			newStmts = append(newStmts, rp)
			continue
		}

		if !endsWithReturn(ifStmt.Then) {
			newStmts = append(newStmts, rp)
			continue
		}

		// Keep the if without else.
		noElse := *ifStmt
		noElse.ElsePart = nil
		newStmts = append(newStmts, tree.RightPadded[tree.Statement]{Element: &noElse, After: rp.After})

		// Extract else body statements as siblings, dedenting by one level.
		dedent := visitor.Init(&dedentElseVisitor{})
		if elseBlock, ok := ifStmt.ElsePart.Element.(*tree.Block); ok {
			for _, s := range elseBlock.Statements {
				dedented := dedent.Visit(s.Element.(tree.Tree), p).(tree.Statement)
				newStmts = append(newStmts, tree.RightPadded[tree.Statement]{Element: dedented, After: s.After})
			}
		}

		changed = true
	}

	if changed {
		return block.WithStatements(newStmts)
	}
	return block
}

// dedentElseVisitor removes one tab from every whitespace in a subtree,
// used to fix indentation when lifting else body statements up one level.
type dedentElseVisitor struct {
	visitor.GoVisitor
}

func (v *dedentElseVisitor) VisitSpace(space tree.Space, p any) tree.Space {
	if strings.Contains(space.Whitespace, "\t") {
		space.Whitespace = strings.Replace(space.Whitespace, "\t", "", 1)
	}
	return space
}

func endsWithReturn(block *tree.Block) bool {
	if block == nil {
		return false
	}
	stmts := block.Statements
	if len(stmts) == 0 {
		return false
	}
	_, isReturn := stmts[len(stmts)-1].Element.(*tree.Return)
	return isReturn
}
