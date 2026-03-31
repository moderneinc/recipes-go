/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package redundancy

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindDoubleDeref finds `*&x` patterns where taking the address of a variable
// and immediately dereferencing it is a no-op. The expression `*&x` is
// equivalent to just `x`.
type FindDoubleDeref struct {
	recipe.Base
}

func (r *FindDoubleDeref) Name() string {
	return "org.openrewrite.golang.codequality.FindDoubleDeref"
}
func (r *FindDoubleDeref) DisplayName() string { return "Find redundant *& (deref of address-of)" }
func (r *FindDoubleDeref) Description() string {
	return "Find `*&x` where taking the address and immediately dereferencing is a no-op."
}
func (r *FindDoubleDeref) Tags() []string { return []string{"cleanup", "redundancy"} }

func (r *FindDoubleDeref) Editor() recipe.TreeVisitor {
	return visitor.Init(&findDoubleDerefVisitor{})
}

type findDoubleDerefVisitor struct {
	visitor.GoVisitor
}

func (v *findDoubleDerefVisitor) VisitUnary(unary *tree.Unary, p any) tree.J {
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

	unary = unary.WithMarkers(
		tree.FoundSearchResult(unary.Markers, "*& is a no-op; dereference of address-of is redundant"),
	)
	return unary
}
