/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
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

func (v *simplifyIfReturnBoolVisitor) VisitBlock(block *java.Block, p any) java.J {
	block = v.GoVisitor.VisitBlock(block, p).(*java.Block)

	stmts := block.Statements
	if len(stmts) == 0 {
		return block
	}

	// Walk the statements looking for the pattern:
	//   if cond { return <bool> }
	//   return <opposite-bool>
	changed := false
	var newStmts []java.RightPadded[java.Statement]

	for i := 0; i < len(stmts); i++ {
		// An `if init; cond` is a golang.StatementWithInit, not a *java.If, so the
		// assertion already excludes init-bearing ifs.
		ifStmt, ok := stmts[i].Element.(*java.If)
		if !ok || ifStmt.Then == nil {
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
					newStmts = append(newStmts, java.RightPadded[java.Statement]{
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
				newStmts = append(newStmts, java.RightPadded[java.Statement]{
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
func singleReturnBool(block *java.Block) (bool, bool) {
	if block == nil || len(block.Statements) != 1 {
		return false, false
	}
	return stmtReturnBool(block.Statements[0].Element)
}

// stmtReturnBool checks if a statement is `return true` or `return false`.
func stmtReturnBool(stmt java.Statement) (bool, bool) {
	ret, ok := stmt.(*java.Return)
	if !ok || ret.Expression == nil {
		return false, false
	}
	ident, ok := ret.Expression.(*java.Identifier)
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
func elseBody(ifStmt *java.If) *java.Block {
	if ifStmt.ElsePart == nil {
		return nil
	}
	if block, ok := ifStmt.ElsePart.Element.(*java.Block); ok {
		return block
	}
	return nil
}

// buildReturn constructs a `return cond` or `return !cond` statement,
// reusing the prefix of the if statement.
func buildReturn(ifStmt *java.If, thenIsTrue bool) *java.Return {
	cond := ifStmt.Condition.Tree.Element
	if !thenIsTrue {
		// Negate the condition: return !cond
		cond = &java.Unary{
			Prefix:   exprPrefix(cond),
			Operator: java.LeftPadded[java.UnaryOperator]{Element: java.Not},
			Operand:  setExprPrefix(cond, java.Space{}),
		}
	}
	return &java.Return{
		Prefix:     ifStmt.Prefix,
		Expression: setExprPrefix(cond, java.SingleSpace),
	}
}
