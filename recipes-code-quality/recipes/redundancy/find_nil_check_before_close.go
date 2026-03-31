/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package redundancy

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindNilCheckBeforeClose finds `if f != nil { f.Close() }` patterns where the
// nil check before Close is suspect. For many types, Close on nil either panics
// or is not meaningful, making this pattern a code smell.
type FindNilCheckBeforeClose struct {
	recipe.Base
}

func (r *FindNilCheckBeforeClose) Name() string {
	return "org.openrewrite.golang.codequality.FindNilCheckBeforeClose"
}
func (r *FindNilCheckBeforeClose) DisplayName() string {
	return "Find nil check before Close"
}
func (r *FindNilCheckBeforeClose) Description() string {
	return "Find `if x != nil { x.Close() }` where the nil check before Close is suspect."
}
func (r *FindNilCheckBeforeClose) Tags() []string {
	return []string{"cleanup", "redundancy"}
}

func (r *FindNilCheckBeforeClose) Editor() recipe.TreeVisitor {
	return visitor.Init(&findNilCheckBeforeCloseVisitor{})
}

type findNilCheckBeforeCloseVisitor struct {
	visitor.GoVisitor
}

func (v *findNilCheckBeforeCloseVisitor) VisitIf(ifStmt *tree.If, p any) tree.J {
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

	ifStmt = ifStmt.WithMarkers(
		tree.FoundSearchResult(ifStmt.Markers, "nil check before Close is suspect"),
	)
	return ifStmt
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
