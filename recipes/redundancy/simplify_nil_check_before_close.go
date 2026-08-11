/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package redundancy

import (
	"github.com/moderneinc/recipes-go/recipes/internal/lstutil"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
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

func (v *simplifyNilCheckBeforeCloseVisitor) VisitIf(ifStmt *java.If, p any) java.J {
	ifStmt = v.GoVisitor.VisitIf(ifStmt, p).(*java.If)

	// Must not have an else clause.
	if ifStmt.ElsePart != nil {
		return ifStmt
	}

	// Must not have an init statement: `if x := ...; cond` is wrapped in a
	// golang.StatementWithInit, so skip Ifs that are its inner statement.
	if lstutil.IsInitWrappedIf(v.Cursor()) {
		return ifStmt
	}

	if ifStmt.Condition == nil {
		return ifStmt
	}

	// Condition must be `x != nil` or `nil != x`.
	varName := nilNotEqualVarName(ifStmt.Condition.Tree.Element)
	if varName == "" {
		return ifStmt
	}

	// The then block must have exactly one statement.
	thenBlock, ok := ifStmt.ThenPart.Element.(*java.Block)
	if !ok || len(thenBlock.Statements) != 1 {
		return ifStmt
	}

	// That single statement must be a MethodInvocation named "Close" on the same variable.
	mi, ok := thenBlock.Statements[0].Element.(*java.MethodInvocation)
	if !ok {
		return ifStmt
	}
	if mi.Name.Name != "Close" {
		return ifStmt
	}
	if mi.Select == nil {
		return ifStmt
	}
	selectIdent, ok := mi.Select.Element.(*java.Identifier)
	if !ok || selectIdent.Name != varName {
		return ifStmt
	}

	// Replace the if statement with just the Close() call, preserving the if's
	// prefix. The leading whitespace lives on the outermost element (the
	// invocation), so carry the if statement's prefix onto it and clear the
	// inner Select element's own (inner-indentation) prefix.
	newMi := *mi
	newMi.Prefix = ifStmt.Prefix
	sel := *newMi.Select
	sel.Element = selectIdent.WithPrefix(java.EmptySpace)
	newMi.Select = &sel
	return &newMi
}

// nilNotEqualVarName extracts the variable name from a `x != nil` or `nil != x`
// condition. Returns "" if the condition does not match.
func nilNotEqualVarName(cond java.Expression) string {
	bin, ok := cond.(*java.Binary)
	if !ok || bin.Operator.Element != java.NotEqual {
		return ""
	}

	leftIdent, leftOk := bin.Left.(*java.Identifier)
	rightIdent, rightOk := bin.Right.(*java.Identifier)

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
