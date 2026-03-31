/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindDeferredClose finds `defer x.Close()` statements. When Close() is
// deferred directly, its error return value is silently discarded. To handle
// the error, wrap the call in a deferred closure that captures the error.
type FindDeferredClose struct {
	recipe.Base
}

func (r *FindDeferredClose) Name() string {
	return "org.openrewrite.golang.codequality.FindDeferredClose"
}
func (r *FindDeferredClose) DisplayName() string { return "Find deferred Close() calls" }
func (r *FindDeferredClose) Description() string {
	return "Find `defer x.Close()` calls where the error return value is silently discarded."
}
func (r *FindDeferredClose) Tags() []string { return []string{"error-handling"} }

func (r *FindDeferredClose) Editor() recipe.TreeVisitor {
	return visitor.Init(&findDeferredCloseVisitor{})
}

type findDeferredCloseVisitor struct {
	visitor.GoVisitor
}

func (v *findDeferredCloseVisitor) VisitDefer(d *tree.Defer, p any) tree.J {
	d = v.GoVisitor.VisitDefer(d, p).(*tree.Defer)

	mi, ok := d.Expr.(*tree.MethodInvocation)
	if !ok {
		return d
	}

	if mi.Name.Name != "Close" {
		return d
	}

	d = d.WithMarkers(
		tree.FoundSearchResult(d.Markers, "deferred Close() silently discards its error"),
	)
	return d
}
