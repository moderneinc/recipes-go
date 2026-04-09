/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification

import "github.com/openrewrite/rewrite/rewrite-go/pkg/tree"

// leadingPrefix returns the effective leading prefix of a binary expression,
// which is on its left child (since Binary.Prefix is typically empty in Go LST).
func leadingPrefix(bin *tree.Binary) tree.Space {
	return exprPrefix(bin.Left)
}

func exprPrefix(expr tree.Expression) tree.Space {
	switch n := expr.(type) {
	case *tree.Identifier:
		return n.Prefix
	case *tree.Literal:
		return n.Prefix
	case *tree.Parentheses:
		return n.Prefix
	case *tree.Unary:
		return n.Prefix
	case *tree.Binary:
		return exprPrefix(n.Left)
	case *tree.FieldAccess:
		return exprPrefix(n.Target)
	case *tree.MethodInvocation:
		if n.Select != nil {
			return exprPrefix(n.Select.Element)
		}
		return exprPrefix(n.Name)
	default:
		return tree.Space{}
	}
}

func setExprPrefix(expr tree.Expression, prefix tree.Space) tree.Expression {
	switch n := expr.(type) {
	case *tree.Identifier:
		return n.WithPrefix(prefix)
	case *tree.Literal:
		return n.WithPrefix(prefix)
	case *tree.Parentheses:
		return n.WithPrefix(prefix)
	case *tree.Unary:
		return &tree.Unary{
			ID: n.ID, Prefix: prefix, Markers: n.Markers,
			Operator: n.Operator, Operand: n.Operand, Type: n.Type,
		}
	case *tree.Binary:
		return &tree.Binary{
			ID: n.ID, Prefix: n.Prefix, Markers: n.Markers,
			Left: setExprPrefix(n.Left, prefix), Operator: n.Operator, Right: n.Right, Type: n.Type,
		}
	case *tree.FieldAccess:
		return n.WithPrefix(prefix)
	case *tree.MethodInvocation:
		return n.WithPrefix(prefix)
	default:
		return expr
	}
}
