/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package redundancy

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// RemoveRedundantInterfaceAssertion removes type assertions to the empty interface
// such as `x.(any)` which are always true and redundant, since every type
// satisfies the empty interface. Replaces `x.(any)` with just `x`.
type RemoveRedundantInterfaceAssertion struct {
	recipe.Base
}

func (r *RemoveRedundantInterfaceAssertion) Name() string {
	return "org.openrewrite.golang.codequality.RemoveRedundantInterfaceAssertion"
}
func (r *RemoveRedundantInterfaceAssertion) DisplayName() string {
	return "Remove redundant type assertion to empty interface"
}
func (r *RemoveRedundantInterfaceAssertion) Description() string {
	return "Remove type assertions to `any` or `interface{}` which are always true and redundant."
}
func (r *RemoveRedundantInterfaceAssertion) Tags() []string {
	return []string{"cleanup", "redundancy"}
}

func (r *RemoveRedundantInterfaceAssertion) Editor() recipe.TreeVisitor {
	return visitor.Init(&removeRedundantInterfaceAssertionVisitor{})
}

type removeRedundantInterfaceAssertionVisitor struct {
	visitor.GoVisitor
}

func (v *removeRedundantInterfaceAssertionVisitor) VisitTypeCast(tc *java.TypeCast, p any) java.J {
	tc = v.GoVisitor.VisitTypeCast(tc, p).(*java.TypeCast)

	if tc.Clazz == nil {
		return tc
	}

	inner := tc.Clazz.Tree.Element

	isRedundant := false

	// Check for `x.(any)` -- the type inside the parentheses is Identifier "any".
	if ident, ok := inner.(*java.Identifier); ok && ident.Name == "any" {
		isRedundant = true
	}

	// Check for `x.(interface{})` -- the type inside is an InterfaceType with an empty body.
	if iface, ok := inner.(*golang.InterfaceType); ok {
		if iface.Body == nil || len(iface.Body.Statements) == 0 {
			isRedundant = true
		}
	}

	if !isRedundant {
		return tc
	}

	// Replace the type assertion with just the inner expression.
	// The Expr already carries the correct prefix (the space between the
	// preceding token and the expression, e.g. the space after "=" in "_ = x.(any)").
	// We prepend tc.Prefix whitespace in case the TypeCast itself had leading space.
	return prependExprPrefix(tc.Expr, tc.Prefix)
}

// prependExprPrefix prepends extra whitespace to an expression's existing prefix.
func prependExprPrefix(expr java.Expression, extra java.Space) java.J {
	if extra.IsEmpty() {
		return expr
	}
	switch n := expr.(type) {
	case *java.Identifier:
		return n.WithPrefix(java.Space{Whitespace: extra.Whitespace + n.Prefix.Whitespace})
	case *java.MethodInvocation:
		return n.WithPrefix(java.Space{Whitespace: extra.Whitespace + n.Prefix.Whitespace})
	case *java.FieldAccess:
		return n.WithPrefix(java.Space{Whitespace: extra.Whitespace + n.Prefix.Whitespace})
	default:
		return expr
	}
}
