/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// AvoidContextWithValue finds calls to `context.WithValue()`. Using context
// values to pass dependencies is considered an anti-pattern because it hides
// function requirements and bypasses compile-time type checking.
type AvoidContextWithValue struct {
	recipe.Base
}

func (r *AvoidContextWithValue) Name() string {
	return "org.openrewrite.golang.codequality.AvoidContextWithValue"
}
func (r *AvoidContextWithValue) DisplayName() string { return "Avoid context.WithValue" }
func (r *AvoidContextWithValue) Description() string {
	return "Find calls to `context.WithValue()`. Context values are an anti-pattern for passing dependencies; prefer explicit function parameters."
}
func (r *AvoidContextWithValue) Tags() []string { return []string{"style"} }

func (r *AvoidContextWithValue) Editor() recipe.TreeVisitor {
	return visitor.Init(&avoidContextWithValueVisitor{})
}

type avoidContextWithValueVisitor struct {
	visitor.GoVisitor
}

func (v *avoidContextWithValueVisitor) VisitMethodInvocation(mi *java.MethodInvocation, p any) java.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*java.MethodInvocation)

	if mi.Select == nil {
		return mi
	}

	ident, ok := mi.Select.Element.(*java.Identifier)
	if !ok || ident.Name != "context" {
		return mi
	}

	if mi.Name.Name != "WithValue" {
		return mi
	}

	mi = mi.WithMarkers(java.MarkupInfo(mi.Markers, "context.WithValue() call; consider passing dependencies explicitly"))
	return mi
}
