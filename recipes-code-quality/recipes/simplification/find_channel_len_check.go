/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindChannelLenCheck finds `len(ch) == 0`, `len(ch) > 0`, and similar
// comparisons on channels. Checking a channel's length is almost always a
// race condition because the value can change between the check and the
// subsequent send/receive.
type FindChannelLenCheck struct {
	recipe.Base
}

func (r *FindChannelLenCheck) Name() string {
	return "org.openrewrite.golang.codequality.FindChannelLenCheck"
}
func (r *FindChannelLenCheck) DisplayName() string { return "Find channel length check" }
func (r *FindChannelLenCheck) Description() string {
	return "Find comparisons on channel length such as `len(ch) == 0`. These are almost always racy because the length can change between the check and the next operation."
}
func (r *FindChannelLenCheck) Tags() []string { return []string{"simplification", "concurrency"} }

func (r *FindChannelLenCheck) Editor() recipe.TreeVisitor {
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
			tree.FoundSearchResult(bin.Markers, "channel length check is racy; the value can change between check and send/receive"),
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
