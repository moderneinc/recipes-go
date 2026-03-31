/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindChannelClose finds calls to the built-in `close()` function. Closing a
// channel should only be done by the sender, and double-closing a channel
// causes a panic. This recipe highlights all close calls for review.
type FindChannelClose struct {
	recipe.Base
}

func (r *FindChannelClose) Name() string {
	return "org.openrewrite.golang.codequality.FindChannelClose"
}
func (r *FindChannelClose) DisplayName() string { return "Find channel close calls" }
func (r *FindChannelClose) Description() string {
	return "Find calls to the built-in `close()` function. Channels should only be closed by the sender, and double-closing causes a panic."
}
func (r *FindChannelClose) Tags() []string { return []string{"style", "concurrency"} }

func (r *FindChannelClose) Editor() recipe.TreeVisitor {
	return visitor.Init(&findChannelCloseVisitor{})
}

type findChannelCloseVisitor struct {
	visitor.GoVisitor
}

func (v *findChannelCloseVisitor) VisitMethodInvocation(mi *tree.MethodInvocation, p any) tree.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*tree.MethodInvocation)

	// Match: close(...) — built-in, so no Select and Name == "close".
	if mi.Select != nil || mi.Name.Name != "close" {
		return mi
	}

	mi = mi.WithMarkers(tree.FoundSearchResult(mi.Markers, "ensure channel is only closed by the sender"))
	return mi
}
