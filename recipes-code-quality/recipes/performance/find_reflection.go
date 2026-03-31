/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package performance

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindReflection finds calls to `reflect.TypeOf()` and `reflect.ValueOf()`.
// Reflection is slow and should be avoided in hot paths.
type FindReflection struct {
	recipe.Base
}

func (r *FindReflection) Name() string {
	return "org.openrewrite.golang.codequality.FindReflection"
}
func (r *FindReflection) DisplayName() string { return "Find reflection usage" }
func (r *FindReflection) Description() string {
	return "Find `reflect.TypeOf()` and `reflect.ValueOf()` calls. Reflection is slow and should be avoided in performance-sensitive code."
}
func (r *FindReflection) Tags() []string { return []string{"performance"} }

func (r *FindReflection) Editor() recipe.TreeVisitor {
	return visitor.Init(&findReflectionVisitor{})
}

type findReflectionVisitor struct {
	visitor.GoVisitor
}

func (v *findReflectionVisitor) VisitMethodInvocation(mi *tree.MethodInvocation, p any) tree.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*tree.MethodInvocation)

	if mi.Select == nil {
		return mi
	}

	ident, ok := mi.Select.Element.(*tree.Identifier)
	if !ok || ident.Name != "reflect" {
		return mi
	}

	if mi.Name.Name != "TypeOf" && mi.Name.Name != "ValueOf" {
		return mi
	}

	mi = mi.WithMarkers(
		tree.FoundSearchResult(mi.Markers, "reflection is slow; avoid in performance-sensitive code"),
	)
	return mi
}
