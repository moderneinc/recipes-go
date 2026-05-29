/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
)

// leadingPrefix returns the effective leading prefix of a binary expression,
// which is on its left child (since Binary.Prefix is typically empty in Go LST).
func leadingPrefix(bin *java.Binary) java.Space {
	return exprPrefix(bin.Left)
}

func exprPrefix(expr java.Expression) java.Space {
	switch n := expr.(type) {
	case *java.Identifier:
		return n.Prefix
	case *java.Literal:
		return n.Prefix
	case *java.Parentheses:
		return n.Prefix
	case *java.Unary:
		return n.Prefix
	case *java.Binary:
		return exprPrefix(n.Left)
	case *java.FieldAccess:
		return exprPrefix(n.Target)
	case *java.MethodInvocation:
		if n.Select != nil {
			return exprPrefix(n.Select.Element)
		}
		return exprPrefix(n.Name)
	default:
		return java.Space{}
	}
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
	case *java.Binary:
		return &java.Binary{
			ID: n.ID, Prefix: n.Prefix, Markers: n.Markers,
			Left: setExprPrefix(n.Left, prefix), Operator: n.Operator, Right: n.Right, Type: n.Type,
		}
	case *java.FieldAccess:
		return n.WithPrefix(prefix)
	case *java.MethodInvocation:
		return n.WithPrefix(prefix)
	default:
		return expr
	}
}
