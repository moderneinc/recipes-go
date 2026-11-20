/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package lstutil

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// IsRepeatableExpression deliberately accepts only expressions whose evaluation
// cannot call user code or receive from a channel. Unknown nodes are refused.
// Repeating a read without intervening effects is stable in race-free Go code.
func IsRepeatableExpression(expr java.Expression) bool {
	switch e := expr.(type) {
	case *java.Identifier, *java.Literal:
		return true
	case *java.Binary:
		return IsRepeatableExpression(e.Left) && IsRepeatableExpression(e.Right)
	case *java.Parentheses:
		return IsRepeatableExpression(e.Tree.Element)
	default:
		return false
	}
}

// HasLabel refuses deletion even for labels nested in another statement. A goto
// outside that subtree may reach it; proving otherwise needs control flow.
func HasLabel(tree java.J) bool {
	scan := &labelScanner{}
	scan.Self = scan
	scan.Visit(tree, nil)
	return scan.found
}

type labelScanner struct {
	visitor.GoVisitor
	found bool
}

func (s *labelScanner) VisitLabel(label *java.Label, p any) java.J {
	s.found = true
	return label
}
