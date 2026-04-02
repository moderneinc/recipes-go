/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// SimplifyIfReturnBool replaces `if cond { return true } return false` and
// `if cond { return true } else { return false }` with `return cond`.
// Also handles the inverted case where the then-block returns false and the
// else/following return returns true, producing `return !cond`.
// Staticcheck: S1008
type SimplifyIfReturnBool struct {
	recipe.Base
}

func (r *SimplifyIfReturnBool) Name() string {
	return "org.openrewrite.golang.codequality.SimplifyIfReturnBool"
}
func (r *SimplifyIfReturnBool) DisplayName() string { return "Simplify if-return-bool" }
func (r *SimplifyIfReturnBool) Description() string {
	return "Replace `if cond { return true }; return false` with `return cond`, and vice versa."
}
func (r *SimplifyIfReturnBool) Tags() []string { return []string{"cleanup", "simplification"} }

func (r *SimplifyIfReturnBool) Editor() recipe.TreeVisitor {
	return visitor.Init(&simplifyIfReturnBoolVisitor{})
}

type simplifyIfReturnBoolVisitor struct {
	visitor.GoVisitor
}

func (v *simplifyIfReturnBoolVisitor) VisitBlock(block *tree.Block, p any) tree.J {
	block = v.GoVisitor.VisitBlock(block, p).(*tree.Block)

	stmts := block.Statements
	if len(stmts) < 2 {
		return block
	}

	// Walk the statements looking for the pattern:
	//   if cond { return <bool> }
	//   return <opposite-bool>
	changed := false
	var newStmts []tree.RightPadded[tree.Statement]

	for i := 0; i < len(stmts); i++ {
		ifStmt, ok := stmts[i].Element.(*tree.If)
		if !ok || ifStmt.Init != nil || ifStmt.Then == nil {
			newStmts = append(newStmts, stmts[i])
			continue
		}

		// Pattern 1: if cond { return true } else { return false }
		// (or if cond { return false } else { return true })
		if ifStmt.ElsePart != nil {
			elseBlock := elseBody(ifStmt)
			if elseBlock != nil {
				thenBool, thenOk := singleReturnBool(ifStmt.Then)
				elseBool, elseOk := singleReturnBool(elseBlock)
				if thenOk && elseOk && thenBool != elseBool {
					ret := buildReturn(ifStmt, thenBool)
					changed = true
					newStmts = append(newStmts, tree.RightPadded[tree.Statement]{
						Element: ret,
						After:   stmts[i].After,
					})
					continue
				}
			}
			newStmts = append(newStmts, stmts[i])
			continue
		}

		// Pattern 2: if cond { return true } return false
		// (or if cond { return false } return true)
		if ifStmt.ElsePart == nil && i+1 < len(stmts) {
			thenBool, thenOk := singleReturnBool(ifStmt.Then)
			nextBool, nextOk := stmtReturnBool(stmts[i+1].Element)
			if thenOk && nextOk && thenBool != nextBool {
				ret := buildReturn(ifStmt, thenBool)
				changed = true
				newStmts = append(newStmts, tree.RightPadded[tree.Statement]{
					Element: ret,
					After:   stmts[i+1].After,
				})
				i++ // skip the next return statement
				continue
			}
		}

		newStmts = append(newStmts, stmts[i])
	}

	if !changed {
		return block
	}
	return block.WithStatements(newStmts)
}

// singleReturnBool checks if a block contains exactly one statement that is
// `return true` or `return false`. Returns the boolean value and true if matched.
func singleReturnBool(block *tree.Block) (bool, bool) {
	if block == nil || len(block.Statements) != 1 {
		return false, false
	}
	return stmtReturnBool(block.Statements[0].Element)
}

// stmtReturnBool checks if a statement is `return true` or `return false`.
func stmtReturnBool(stmt tree.Statement) (bool, bool) {
	ret, ok := stmt.(*tree.Return)
	if !ok || len(ret.Expressions) != 1 {
		return false, false
	}
	ident, ok := ret.Expressions[0].Element.(*tree.Identifier)
	if !ok {
		return false, false
	}
	switch ident.Name {
	case "true":
		return true, true
	case "false":
		return false, true
	}
	return false, false
}

// elseBody extracts the Block from an if-else clause.
func elseBody(ifStmt *tree.If) *tree.Block {
	if ifStmt.ElsePart == nil {
		return nil
	}
	elsePart := ifStmt.ElsePart.Element
	if elseStmt, ok := elsePart.(*tree.Else); ok {
		if block, ok := elseStmt.Body.Element.(*tree.Block); ok {
			return block
		}
	}
	return nil
}

// buildReturn constructs a `return cond` or `return !cond` statement,
// reusing the prefix of the if statement.
func buildReturn(ifStmt *tree.If, thenIsTrue bool) *tree.Return {
	cond := ifStmt.Condition
	if !thenIsTrue {
		// Negate the condition: return !cond
		cond = &tree.Unary{
			Prefix:   exprPrefix(cond),
			Operator: tree.LeftPadded[tree.UnaryOperator]{Element: tree.Not},
			Operand:  setExprPrefix(cond, tree.Space{}),
		}
	}
	return &tree.Return{
		Prefix: ifStmt.Prefix,
		Expressions: []tree.RightPadded[tree.Expression]{
			{Element: setExprPrefix(cond, tree.SingleSpace)},
		},
	}
}
