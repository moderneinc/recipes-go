/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// CheckContextError finds `ctx.Err()` calls. The context error should be inspected
// to distinguish between cancellation and deadline exceeded.
type CheckContextError struct {
	recipe.Base
}

func (r *CheckContextError) Name() string {
	return "org.openrewrite.golang.codequality.CheckContextError"
}
func (r *CheckContextError) DisplayName() string { return "Check context error" }
func (r *CheckContextError) Description() string {
	return "Find `ctx.Err()` calls. The context error should be inspected to distinguish between cancellation and deadline exceeded."
}
func (r *CheckContextError) Tags() []string { return []string{"error-handling", "lint"} }

func (r *CheckContextError) Editor() recipe.TreeVisitor {
	return visitor.Init(&checkContextErrorVisitor{})
}

type checkContextErrorVisitor struct {
	visitor.GoVisitor
}

func (v *checkContextErrorVisitor) VisitMethodInvocation(mi *java.MethodInvocation, p any) java.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*java.MethodInvocation)

	if mi.Select == nil {
		return mi
	}

	ident, ok := mi.Select.Element.(*java.Identifier)
	if !ok || ident.Name != "ctx" {
		return mi
	}

	if mi.Name.Name != "Err" {
		return mi
	}

	mi = mi.WithMarkers(
		java.MarkupInfo(mi.Markers, "ctx.Err() found; inspect the context error"),
	)
	return mi
}
