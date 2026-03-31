/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindCloseCall finds calls to `Close()` methods. The error returned by
// Close() is frequently ignored, which can hide data-loss bugs (e.g. when
// writing buffered data to a file).
type FindCloseCall struct {
	recipe.Base
}

func (r *FindCloseCall) Name() string {
	return "org.openrewrite.golang.codequality.FindCloseCall"
}
func (r *FindCloseCall) DisplayName() string { return "Find Close() calls" }
func (r *FindCloseCall) Description() string {
	return "Find calls to `Close()` methods whose error return value may be ignored."
}
func (r *FindCloseCall) Tags() []string { return []string{"error-handling"} }

func (r *FindCloseCall) Editor() recipe.TreeVisitor {
	return visitor.Init(&findCloseCallVisitor{})
}

type findCloseCallVisitor struct {
	visitor.GoVisitor
}

func (v *findCloseCallVisitor) VisitMethodInvocation(mi *tree.MethodInvocation, p any) tree.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*tree.MethodInvocation)

	// Match: x.Close() — any method named "Close" with a receiver.
	if mi.Select == nil || mi.Name.Name != "Close" {
		return mi
	}

	mi = mi.WithMarkers(
		tree.FoundSearchResult(mi.Markers, "ensure Close() error is handled"),
	)
	return mi
}
