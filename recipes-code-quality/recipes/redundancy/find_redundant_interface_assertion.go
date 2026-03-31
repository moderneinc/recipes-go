/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package redundancy

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindRedundantInterfaceAssertion finds type assertions to the empty interface
// such as `x.(any)` which are always true and redundant, since every type
// satisfies the empty interface.
type FindRedundantInterfaceAssertion struct {
	recipe.Base
}

func (r *FindRedundantInterfaceAssertion) Name() string {
	return "org.openrewrite.golang.codequality.FindRedundantInterfaceAssertion"
}
func (r *FindRedundantInterfaceAssertion) DisplayName() string {
	return "Find redundant type assertion to empty interface"
}
func (r *FindRedundantInterfaceAssertion) Description() string {
	return "Find type assertions to `any` or `interface{}` which are always true and redundant."
}
func (r *FindRedundantInterfaceAssertion) Tags() []string {
	return []string{"cleanup", "redundancy"}
}

func (r *FindRedundantInterfaceAssertion) Editor() recipe.TreeVisitor {
	return visitor.Init(&findRedundantInterfaceAssertionVisitor{})
}

type findRedundantInterfaceAssertionVisitor struct {
	visitor.GoVisitor
}

func (v *findRedundantInterfaceAssertionVisitor) VisitTypeCast(tc *tree.TypeCast, p any) tree.J {
	tc = v.GoVisitor.VisitTypeCast(tc, p).(*tree.TypeCast)

	if tc.Clazz == nil {
		return tc
	}

	inner := tc.Clazz.Tree.Element

	// Check for `x.(any)` — the type inside the parentheses is Identifier "any".
	if ident, ok := inner.(*tree.Identifier); ok && ident.Name == "any" {
		tc = tc.WithMarkers(
			tree.FoundSearchResult(tc.Markers, "type assertion to empty interface is redundant"),
		)
		return tc
	}

	// Check for `x.(interface{})` — the type inside is an InterfaceType with an empty body.
	if iface, ok := inner.(*tree.InterfaceType); ok {
		if iface.Body == nil || len(iface.Body.Statements) == 0 {
			tc = tc.WithMarkers(
				tree.FoundSearchResult(tc.Markers, "type assertion to empty interface is redundant"),
			)
			return tc
		}
	}

	return tc
}
