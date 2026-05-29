/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package redundancy

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
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

func (v *removeDoubleDerefVisitor) VisitUnary(unary *java.Unary, p any) java.J {
	unary = v.GoVisitor.VisitUnary(unary, p).(*java.Unary)

	// Outer operator must be Deref (*).
	if unary.Operator.Element != java.Deref {
		return unary
	}

	// Operand must be another Unary with operator AddressOf (&).
	inner, ok := unary.Operand.(*java.Unary)
	if !ok || inner.Operator.Element != java.AddressOf {
		return unary
	}

	// Replace *&x with x, preserving the outer unary's prefix.
	switch operand := inner.Operand.(type) {
	case *java.Identifier:
		return operand.WithPrefix(unary.Prefix)
	case *java.MethodInvocation:
		return operand.WithPrefix(unary.Prefix)
	case *java.FieldAccess:
		return operand.WithPrefix(unary.Prefix)
	case *java.Literal:
		return operand.WithPrefix(unary.Prefix)
	case *java.Parentheses:
		return operand.WithPrefix(unary.Prefix)
	default:
		return inner.Operand
	}
}
