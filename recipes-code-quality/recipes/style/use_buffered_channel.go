/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// UseBufferedChannel finds `make(chan T)` calls without a buffer size.
// Unbuffered channels are fine in many cases but worth flagging for review,
// as they block until both sender and receiver are ready.
type UseBufferedChannel struct {
	recipe.Base
}

func (r *UseBufferedChannel) Name() string {
	return "org.openrewrite.golang.codequality.UseBufferedChannel"
}
func (r *UseBufferedChannel) DisplayName() string { return "Use buffered channel" }
func (r *UseBufferedChannel) Description() string {
	return "Find `make(chan T)` calls without a buffer size. Unbuffered channels block until both sender and receiver are ready."
}
func (r *UseBufferedChannel) Tags() []string { return []string{"concurrency"} }

func (r *UseBufferedChannel) Editor() recipe.TreeVisitor {
	return visitor.Init(&useBufferedChannelVisitor{})
}

type useBufferedChannelVisitor struct {
	visitor.GoVisitor
}

func (v *useBufferedChannelVisitor) VisitMethodInvocation(mi *tree.MethodInvocation, p any) tree.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*tree.MethodInvocation)

	// Must be a free-function call to "make" (no receiver).
	if mi.Select != nil || mi.Name == nil || mi.Name.Name != "make" {
		return mi
	}

	// Count real arguments (skip Empty sentinels).
	var realArgs []tree.Expression
	for _, arg := range mi.Arguments.Elements {
		if _, isEmpty := arg.Element.(*tree.Empty); !isEmpty {
			realArgs = append(realArgs, arg.Element)
		}
	}

	// make(chan T) has exactly 1 argument; make(chan T, size) has 2.
	if len(realArgs) != 1 {
		return mi
	}

	// The single argument must be a channel type.
	if _, isChan := realArgs[0].(*tree.Channel); !isChan {
		return mi
	}

	mi = mi.WithMarkers(tree.MarkupInfo(mi.Markers, "unbuffered channel"))
	return mi
}
