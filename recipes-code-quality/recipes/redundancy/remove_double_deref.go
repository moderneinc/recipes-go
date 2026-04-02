/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package redundancy

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// RemoveDoubleDeref removes `*&x` patterns where taking the address of a variable
// and immediately dereferencing it is a no-op. The expression `*&x` is
// replaced with just `x`.
type RemoveDoubleDeref struct {
	recipe.Base
}

func (r *RemoveDoubleDeref) Name() string {
	return "org.openrewrite.golang.codequality.RemoveDoubleDeref"
}
func (r *RemoveDoubleDeref) DisplayName() string { return "Remove redundant *& (deref of address-of)" }
func (r *RemoveDoubleDeref) Description() string {
	return "Remove `*&x` where taking the address and immediately dereferencing is a no-op."
}
func (r *RemoveDoubleDeref) Tags() []string { return []string{"cleanup", "redundancy"} }

func (r *RemoveDoubleDeref) Editor() recipe.TreeVisitor {
	return visitor.Init(&removeDoubleDerefVisitor{})
}

type removeDoubleDerefVisitor struct {
	visitor.GoVisitor
}

func (v *removeDoubleDerefVisitor) VisitUnary(unary *tree.Unary, p any) tree.J {
	unary = v.GoVisitor.VisitUnary(unary, p).(*tree.Unary)

	// Outer operator must be Deref (*).
	if unary.Operator.Element != tree.Deref {
		return unary
	}

	// Operand must be another Unary with operator AddressOf (&).
	inner, ok := unary.Operand.(*tree.Unary)
	if !ok || inner.Operator.Element != tree.AddressOf {
		return unary
	}

	// Replace *&x with x, preserving the outer unary's prefix.
	switch operand := inner.Operand.(type) {
	case *tree.Identifier:
		return operand.WithPrefix(unary.Prefix)
	case *tree.MethodInvocation:
		return operand.WithPrefix(unary.Prefix)
	case *tree.FieldAccess:
		return operand.WithPrefix(unary.Prefix)
	case *tree.Literal:
		return operand.WithPrefix(unary.Prefix)
	case *tree.Parentheses:
		return operand.WithPrefix(unary.Prefix)
	default:
		return inner.Operand
	}
}
