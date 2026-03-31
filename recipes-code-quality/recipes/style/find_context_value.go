/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindContextWithValue finds calls to `context.WithValue()`. Using context
// values to pass dependencies is considered an anti-pattern because it hides
// function requirements and bypasses compile-time type checking.
type FindContextWithValue struct {
	recipe.Base
}

func (r *FindContextWithValue) Name() string {
	return "org.openrewrite.golang.codequality.FindContextWithValue"
}
func (r *FindContextWithValue) DisplayName() string { return "Find context.WithValue() calls" }
func (r *FindContextWithValue) Description() string {
	return "Find calls to `context.WithValue()`. Context values are an anti-pattern for passing dependencies; prefer explicit function parameters."
}
func (r *FindContextWithValue) Tags() []string { return []string{"style"} }

func (r *FindContextWithValue) Editor() recipe.TreeVisitor {
	return visitor.Init(&findContextWithValueVisitor{})
}

type findContextWithValueVisitor struct {
	visitor.GoVisitor
}

func (v *findContextWithValueVisitor) VisitMethodInvocation(mi *tree.MethodInvocation, p any) tree.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*tree.MethodInvocation)

	if mi.Select == nil {
		return mi
	}

	ident, ok := mi.Select.Element.(*tree.Identifier)
	if !ok || ident.Name != "context" {
		return mi
	}

	if mi.Name.Name != "WithValue" {
		return mi
	}

	mi = mi.WithMarkers(tree.FoundSearchResult(mi.Markers, "context.WithValue() call; consider passing dependencies explicitly"))
	return mi
}
