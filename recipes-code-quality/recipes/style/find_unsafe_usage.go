/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindUnsafeUsage finds any usage of the `unsafe` package. The unsafe package
// bypasses Go's type safety guarantees and should be avoided unless absolutely
// necessary. Common uses include unsafe.Pointer, unsafe.Sizeof, and
// unsafe.Alignof.
type FindUnsafeUsage struct {
	recipe.Base
}

func (r *FindUnsafeUsage) Name() string {
	return "org.openrewrite.golang.codequality.FindUnsafeUsage"
}
func (r *FindUnsafeUsage) DisplayName() string { return "Find unsafe package usage" }
func (r *FindUnsafeUsage) Description() string {
	return "Find any usage of the `unsafe` package. The unsafe package bypasses Go's type safety guarantees and should be avoided unless absolutely necessary."
}
func (r *FindUnsafeUsage) Tags() []string { return []string{"security"} }

func (r *FindUnsafeUsage) Editor() recipe.TreeVisitor {
	return visitor.Init(&findUnsafeUsageVisitor{})
}

type findUnsafeUsageVisitor struct {
	visitor.GoVisitor
}

func (v *findUnsafeUsageVisitor) VisitFieldAccess(fa *tree.FieldAccess, p any) tree.J {
	fa = v.GoVisitor.VisitFieldAccess(fa, p).(*tree.FieldAccess)

	ident, ok := fa.Target.(*tree.Identifier)
	if !ok || ident.Name != "unsafe" {
		return fa
	}

	fa = fa.WithMarkers(tree.FoundSearchResult(fa.Markers, "unsafe package usage"))
	return fa
}

func (v *findUnsafeUsageVisitor) VisitMethodInvocation(mi *tree.MethodInvocation, p any) tree.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*tree.MethodInvocation)

	if mi.Select == nil {
		return mi
	}

	ident, ok := mi.Select.Element.(*tree.Identifier)
	if !ok || ident.Name != "unsafe" {
		return mi
	}

	mi = mi.WithMarkers(tree.FoundSearchResult(mi.Markers, "unsafe package usage"))
	return mi
}
