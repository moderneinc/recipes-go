/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification

import (
	"strings"

	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// MergeCollapsibleIf merges nested if statements into a single if with
// a combined condition using &&. For example:
//
//	if a {
//	    if b { x }
//	}
//
// becomes:
//
//	if a && b { x }
//
// Neither if may have an else clause, the outer if's body must contain
// only the inner if, and neither if may have an init statement.
type MergeCollapsibleIf struct {
	recipe.Base
}

func (r *MergeCollapsibleIf) Name() string {
	return "org.openrewrite.golang.codequality.MergeCollapsibleIf"
}
func (r *MergeCollapsibleIf) DisplayName() string { return "Merge collapsible if statements" }
func (r *MergeCollapsibleIf) Description() string {
	return "Merge nested `if` statements into a single `if` with `&&` when neither has an else clause and the outer body contains only the inner `if`."
}
func (r *MergeCollapsibleIf) Tags() []string { return []string{"cleanup", "simplification"} }

func (r *MergeCollapsibleIf) Editor() recipe.TreeVisitor {
	return visitor.Init(&mergeCollapsibleIfVisitor{})
}

type mergeCollapsibleIfVisitor struct {
	visitor.GoVisitor
}

func (v *mergeCollapsibleIfVisitor) VisitIf(ifStmt *java.If, p any) java.J {
	ifStmt = v.GoVisitor.VisitIf(ifStmt, p).(*java.If)

	// Outer if must not have an else clause or init statement.
	if ifStmt.ElsePart != nil || ifStmt.Init != nil {
		return ifStmt
	}

	// Body must contain exactly one statement.
	if ifStmt.Then == nil || len(ifStmt.Then.Statements) != 1 {
		return ifStmt
	}

	// That single statement must be another if.
	innerIf, ok := ifStmt.Then.Statements[0].Element.(*java.If)
	if !ok {
		return ifStmt
	}

	// Inner if must not have an else clause or init statement.
	if innerIf.ElsePart != nil || innerIf.Init != nil {
		return ifStmt
	}

	// Build the combined condition: outerCond && innerCond.
	// Wrap either side in parentheses if it is a || expression to preserve precedence.
	outerCond := maybeWrapOr(ifStmt.Condition)
	innerCond := maybeWrapOr(innerIf.Condition)

	combined := &java.Binary{
		Left:     setExprPrefix(outerCond, exprPrefix(ifStmt.Condition)),
		Operator: java.LeftPadded[java.BinaryOperator]{Before: java.SingleSpace, Element: java.LogicalAnd},
		Right:    setExprPrefix(innerCond, java.SingleSpace),
	}

	// Dedent the inner body by one level since it's moving up.
	dedent := visitor.Init(&dedentCollapsedVisitor{})
	newBody := dedent.Visit(innerIf.Then, p).(*java.Block)

	return ifStmt.WithCondition(combined).WithThen(newBody)
}

// dedentCollapsedVisitor removes one tab from every whitespace in a subtree,
// used to fix indentation when collapsing nested if statements.
type dedentCollapsedVisitor struct {
	visitor.GoVisitor
}

func (v *dedentCollapsedVisitor) VisitSpace(space java.Space, p any) java.Space {
	if strings.Contains(space.Whitespace, "\t") {
		space.Whitespace = strings.Replace(space.Whitespace, "\t", "", 1)
	}
	return space
}

// maybeWrapOr wraps the expression in parentheses if it is a top-level || binary,
// because && binds tighter than || in Go.
func maybeWrapOr(expr java.Expression) java.Expression {
	bin, ok := expr.(*java.Binary)
	if !ok || bin.Operator.Element != java.LogicalOr {
		return expr
	}
	return &java.Parentheses{
		Prefix: exprPrefix(expr),
		Tree: java.RightPadded[java.Expression]{
			Element: setExprPrefix(expr, java.Space{}),
		},
	}
}
