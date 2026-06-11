/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package performance

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// AvoidReflection finds calls to `reflect.TypeOf()` and `reflect.ValueOf()`.
// Reflection is slow and should be avoided in hot paths.
type AvoidReflection struct {
	recipe.Base
}

func (r *AvoidReflection) Name() string {
	return "org.openrewrite.golang.codequality.AvoidReflection"
}
func (r *AvoidReflection) DisplayName() string { return "Avoid reflection" }
func (r *AvoidReflection) Description() string {
	return "Find `reflect.TypeOf()` and `reflect.ValueOf()` calls. Reflection is slow and should be avoided in performance-sensitive code."
}
func (r *AvoidReflection) Tags() []string { return []string{"performance"} }

func (r *AvoidReflection) Editor() recipe.TreeVisitor {
	return visitor.Init(&avoidReflectionVisitor{})
}

type avoidReflectionVisitor struct {
	visitor.GoVisitor
}

func (v *avoidReflectionVisitor) VisitMethodInvocation(mi *java.MethodInvocation, p any) java.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*java.MethodInvocation)

	if mi.Select == nil {
		return mi
	}

	ident, ok := mi.Select.Element.(*java.Identifier)
	if !ok || ident.Name != "reflect" {
		return mi
	}

	if mi.Name.Name != "TypeOf" && mi.Name.Name != "ValueOf" {
		return mi
	}

	mi = mi.WithMarkers(
		java.MarkupInfo(mi.Markers, "reflection is slow; avoid in performance-sensitive code"),
	)
	return mi
}
