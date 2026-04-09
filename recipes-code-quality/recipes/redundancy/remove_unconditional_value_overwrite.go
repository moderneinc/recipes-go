/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package redundancy

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/printer"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// RemoveUnconditionalValueOverwrite removes consecutive assignments to the same
// map key (or slice index) within the same block when the first value is
// immediately overwritten by the second. The first assignment is dead code.
//
//	m["key"] = 1
//	m["key"] = 2
//	// becomes
//	m["key"] = 2
type RemoveUnconditionalValueOverwrite struct {
	recipe.Base
}

func (r *RemoveUnconditionalValueOverwrite) Name() string {
	return "org.openrewrite.golang.codequality.RemoveUnconditionalValueOverwrite"
}
func (r *RemoveUnconditionalValueOverwrite) DisplayName() string {
	return "Remove unconditional value overwrite"
}
func (r *RemoveUnconditionalValueOverwrite) Description() string {
	return "Remove consecutive assignments to the same collection key or index where " +
		"the first value is immediately overwritten and never read."
}
func (r *RemoveUnconditionalValueOverwrite) Tags() []string {
	return []string{"cleanup", "redundancy", "RSPEC-S4143"}
}

func (r *RemoveUnconditionalValueOverwrite) Editor() recipe.TreeVisitor {
	return visitor.Init(&removeUnconditionalValueOverwriteVisitor{})
}

type removeUnconditionalValueOverwriteVisitor struct {
	visitor.GoVisitor
}

func (v *removeUnconditionalValueOverwriteVisitor) VisitBlock(block *tree.Block, p any) tree.J {
	block = v.GoVisitor.VisitBlock(block, p).(*tree.Block)

	stmts := block.Statements
	if len(stmts) < 2 {
		return block
	}

	changed := false
	var result []tree.RightPadded[tree.Statement]

	for i := 0; i < len(stmts); i++ {
		// Look ahead: if this and the next statement both assign to the same
		// indexed target, skip this (dead) assignment.
		if i+1 < len(stmts) && isOverwrittenIndexAssignment(stmts[i], stmts[i+1]) {
			changed = true
			continue
		}
		result = append(result, stmts[i])
	}

	if !changed {
		return block
	}
	return block.WithStatements(result)
}

// isOverwrittenIndexAssignment returns true when both statements are
// assignments of the form `receiver[key] = value` with the same receiver
// and key (printed form, ignoring whitespace).
func isOverwrittenIndexAssignment(first, second tree.RightPadded[tree.Statement]) bool {
	a := extractIndexAssignment(first)
	b := extractIndexAssignment(second)
	if a == nil || b == nil {
		return false
	}
	return printNorm(a.Indexed) == printNorm(b.Indexed) &&
		printNorm(a.Dimension.Index.Element) == printNorm(b.Dimension.Index.Element)
}

// extractIndexAssignment checks whether a statement is an assignment whose
// left-hand side is an ArrayAccess (index expression), and returns the
// ArrayAccess if so.
func extractIndexAssignment(stmt tree.RightPadded[tree.Statement]) *tree.ArrayAccess {
	assign, ok := stmt.Element.(*tree.Assignment)
	if !ok {
		return nil
	}
	aa, ok := assign.Variable.(*tree.ArrayAccess)
	if !ok {
		return nil
	}
	return aa
}

func printNorm(node tree.J) string {
	return printer.Print(node)
}
