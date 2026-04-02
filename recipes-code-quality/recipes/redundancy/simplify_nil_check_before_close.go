/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package redundancy

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// SimplifyNilCheckBeforeClose replaces `if f != nil { f.Close() }` with just
// `f.Close()`. The nil check before Close is redundant for most types.
type SimplifyNilCheckBeforeClose struct {
	recipe.Base
}

func (r *SimplifyNilCheckBeforeClose) Name() string {
	return "org.openrewrite.golang.codequality.SimplifyNilCheckBeforeClose"
}
func (r *SimplifyNilCheckBeforeClose) DisplayName() string {
	return "Simplify nil check before Close"
}
func (r *SimplifyNilCheckBeforeClose) Description() string {
	return "Replace `if x != nil { x.Close() }` with `x.Close()` where the nil check is redundant."
}
func (r *SimplifyNilCheckBeforeClose) Tags() []string {
	return []string{"cleanup", "redundancy"}
}

func (r *SimplifyNilCheckBeforeClose) Editor() recipe.TreeVisitor {
	return visitor.Init(&simplifyNilCheckBeforeCloseVisitor{})
}

type simplifyNilCheckBeforeCloseVisitor struct {
	visitor.GoVisitor
}

func (v *simplifyNilCheckBeforeCloseVisitor) VisitIf(ifStmt *tree.If, p any) tree.J {
	ifStmt = v.GoVisitor.VisitIf(ifStmt, p).(*tree.If)

	// Must not have an else clause.
	if ifStmt.ElsePart != nil {
		return ifStmt
	}

	// Must not have an init statement.
	if ifStmt.Init != nil {
		return ifStmt
	}

	// Condition must be `x != nil` or `nil != x`.
	varName := nilNotEqualVarName(ifStmt.Condition)
	if varName == "" {
		return ifStmt
	}

	// The then block must have exactly one statement.
	if ifStmt.Then == nil || len(ifStmt.Then.Statements) != 1 {
		return ifStmt
	}

	// That single statement must be a MethodInvocation named "Close" on the same variable.
	mi, ok := ifStmt.Then.Statements[0].Element.(*tree.MethodInvocation)
	if !ok {
		return ifStmt
	}
	if mi.Name.Name != "Close" {
		return ifStmt
	}
	if mi.Select == nil {
		return ifStmt
	}
	selectIdent, ok := mi.Select.Element.(*tree.Identifier)
	if !ok || selectIdent.Name != varName {
		return ifStmt
	}

	// Replace the if statement with just the Close() call, preserving the if's prefix.
	// The visible leading whitespace of `f.Close()` lives on the Select element (the `f`
	// identifier), not on the MethodInvocation itself. Set it to the if statement's prefix.
	newMi := *mi
	sel := *newMi.Select
	sel.Element = selectIdent.WithPrefix(ifStmt.Prefix)
	newMi.Select = &sel
	return &newMi
}

// nilNotEqualVarName extracts the variable name from a `x != nil` or `nil != x`
// condition. Returns "" if the condition does not match.
func nilNotEqualVarName(cond tree.Expression) string {
	bin, ok := cond.(*tree.Binary)
	if !ok || bin.Operator.Element != tree.NotEqual {
		return ""
	}

	leftIdent, leftOk := bin.Left.(*tree.Identifier)
	rightIdent, rightOk := bin.Right.(*tree.Identifier)

	// x != nil
	if leftOk && rightOk && rightIdent.Name == "nil" {
		return leftIdent.Name
	}
	// nil != x
	if leftOk && rightOk && leftIdent.Name == "nil" {
		return rightIdent.Name
	}
	return ""
}

