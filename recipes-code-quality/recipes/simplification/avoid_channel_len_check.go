/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// AvoidChannelLenCheck finds `len(ch) == 0`, `len(ch) > 0`, and similar
// comparisons on channels. Checking a channel's length is almost always a
// race condition because the value can change between the check and the
// subsequent send/receive.
type AvoidChannelLenCheck struct {
	recipe.Base
}

func (r *AvoidChannelLenCheck) Name() string {
	return "org.openrewrite.golang.codequality.AvoidChannelLenCheck"
}
func (r *AvoidChannelLenCheck) DisplayName() string { return "Avoid channel length check" }
func (r *AvoidChannelLenCheck) Description() string {
	return "Find comparisons on channel length such as `len(ch) == 0`. These are almost always racy because the length can change between the check and the next operation."
}
func (r *AvoidChannelLenCheck) Tags() []string { return []string{"simplification", "concurrency"} }

func (r *AvoidChannelLenCheck) Editor() recipe.TreeVisitor {
	return visitor.Init(&findChannelLenCheckVisitor{})
}

type findChannelLenCheckVisitor struct {
	visitor.GoVisitor
}

func (v *findChannelLenCheckVisitor) VisitBinary(bin *tree.Binary, p any) tree.J {
	bin = v.GoVisitor.VisitBinary(bin, p).(*tree.Binary)

	// Match patterns like: len(ch) == 0, len(ch) > 0, len(ch) != 0, etc.
	if !isComparisonOp(bin.Operator.Element) {
		return bin
	}

	if isLenCall(bin.Left) || isLenCall(bin.Right) {
		bin = bin.WithMarkers(
			tree.MarkupWarn(bin.Markers, "channel length check is racy; the value can change between check and send/receive"),
		)
	}

	return bin
}

// isLenCall checks if the expression is a call to the built-in `len` function.
func isLenCall(expr tree.Expression) bool {
	mi, ok := expr.(*tree.MethodInvocation)
	if !ok {
		return false
	}
	return mi.Select == nil && mi.Name.Name == "len"
}

// isComparisonOp returns true for ==, !=, <, >, <=, >=.
func isComparisonOp(op tree.BinaryOperator) bool {
	switch op {
	case tree.Equal, tree.NotEqual,
		tree.LessThan, tree.GreaterThan,
		tree.LessThanOrEqual, tree.GreaterThanOrEqual:
		return true
	default:
		return false
	}
}
