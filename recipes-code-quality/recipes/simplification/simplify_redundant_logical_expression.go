/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/printer"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// SimplifyRedundantLogicalExpression simplifies binary expressions where both
// operands are identical and the operator is a logical or bitwise and/or.
// For example `x && x` becomes `x`, and `x || x` becomes `x`.
// Arithmetic operators like `-` or `+` and comparisons like `==` are excluded
// because identical operands there may signal a real bug rather than simple
// redundancy.
type SimplifyRedundantLogicalExpression struct {
	recipe.Base
}

func (r *SimplifyRedundantLogicalExpression) Name() string {
	return "org.openrewrite.golang.codequality.SimplifyRedundantLogicalExpression"
}
func (r *SimplifyRedundantLogicalExpression) DisplayName() string {
	return "Simplify redundant logical expression"
}
func (r *SimplifyRedundantLogicalExpression) Description() string {
	return "Simplify `x && x` to `x`, `x || x` to `x`, and similarly for `&` and `|`, " +
		"where both sides of a logical or bitwise operator are identical."
}
func (r *SimplifyRedundantLogicalExpression) Tags() []string {
	return []string{"cleanup", "simplification", "RSPEC-S1764"}
}

func (r *SimplifyRedundantLogicalExpression) Editor() recipe.TreeVisitor {
	return visitor.Init(&simplifyRedundantLogicalExpressionVisitor{})
}

type simplifyRedundantLogicalExpressionVisitor struct {
	visitor.GoVisitor
}

func (v *simplifyRedundantLogicalExpressionVisitor) VisitBinary(bin *java.Binary, p any) java.J {
	bin = v.GoVisitor.VisitBinary(bin, p).(*java.Binary)

	if !isLogicalOrBitwiseOp(bin.Operator.Element) {
		return bin
	}

	// Compare printed representations (ignoring leading whitespace) to determine
	// structural equality of the two operands.
	if printExpr(bin.Left) == printExpr(bin.Right) {
		// Replace the whole binary with just the left operand, preserving the
		// outer prefix so the surrounding code keeps its whitespace.
		return setExprPrefix(bin.Left, exprPrefix(bin.Left))
	}

	return bin
}

func isLogicalOrBitwiseOp(op java.BinaryOperator) bool {
	switch op {
	case java.LogicalAnd, java.LogicalOr, java.BitwiseAnd, java.BitwiseOr:
		return true
	}
	return false
}

// printExpr returns a normalised text representation of an expression for
// equality comparison. Leading whitespace is trimmed so that formatting
// differences between the left and right operands do not cause false negatives.
func printExpr(expr java.Expression) string {
	return printer.Print(setExprPrefix(expr, java.Space{}))
}
