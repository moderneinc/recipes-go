/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
)

// leadingPrefix returns the leading prefix of a binary expression. The parser
// attaches inter-element whitespace to the outermost element, so it lives
// directly on the binary's own prefix.
func leadingPrefix(bin *java.Binary) java.Space {
	return bin.Prefix
}

// exprPrefix returns the node's own leading whitespace. The parser attaches
// inter-element whitespace to the outermost element, so the leading prefix
// lives directly on the node.
func exprPrefix(expr java.Expression) java.Space {
	return expr.GetPrefix()
}

func setExprPrefix(expr java.Expression, prefix java.Space) java.Expression {
	switch n := expr.(type) {
	case *java.Identifier:
		return n.WithPrefix(prefix)
	case *java.Literal:
		return n.WithPrefix(prefix)
	case *java.Parentheses:
		return n.WithPrefix(prefix)
	case *java.Unary:
		return &java.Unary{
			ID: n.ID, Prefix: prefix, Markers: n.Markers,
			Operator: n.Operator, Operand: n.Operand, Type: n.Type,
		}
	case *golang.Unary:
		return n.WithPrefix(prefix)
	case *java.Binary:
		// Put the leading whitespace on the binary (the outermost element) and
		// clear any prefix the leftmost operand still carries, so the leading
		// whitespace isn't doubled.
		return &java.Binary{
			ID: n.ID, Prefix: prefix, Markers: n.Markers,
			Left: setExprPrefix(n.Left, java.Space{}), Operator: n.Operator, Right: n.Right, Type: n.Type,
		}
	case *java.FieldAccess:
		return n.WithPrefix(prefix)
	case *java.MethodInvocation:
		return n.WithPrefix(prefix)
	default:
		return expr
	}
}
