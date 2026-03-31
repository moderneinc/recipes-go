/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package performance

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindChannelCreateInLoop finds `make(chan ...)` calls inside for/range loops.
// Creating a channel on every iteration is usually unnecessary; the channel
// should be created once before the loop.
type FindChannelCreateInLoop struct {
	recipe.Base
}

func (r *FindChannelCreateInLoop) Name() string {
	return "org.openrewrite.golang.codequality.FindChannelCreateInLoop"
}
func (r *FindChannelCreateInLoop) DisplayName() string { return "Find channel creation in loop" }
func (r *FindChannelCreateInLoop) Description() string {
	return "Find `make(chan ...)` calls inside for/range loops. Channel creation in loops suggests the channel should be created once before the loop."
}
func (r *FindChannelCreateInLoop) Tags() []string { return []string{"performance"} }

func (r *FindChannelCreateInLoop) Editor() recipe.TreeVisitor {
	return visitor.Init(&findChannelCreateInLoopVisitor{})
}

type findChannelCreateInLoopVisitor struct {
	visitor.GoVisitor
	insideLoop int
}

func (v *findChannelCreateInLoopVisitor) VisitForLoop(forLoop *tree.ForLoop, p any) tree.J {
	v.insideLoop++
	forLoop = v.GoVisitor.VisitForLoop(forLoop, p).(*tree.ForLoop)
	v.insideLoop--
	return forLoop
}

func (v *findChannelCreateInLoopVisitor) VisitForEachLoop(forEach *tree.ForEachLoop, p any) tree.J {
	v.insideLoop++
	forEach = v.GoVisitor.VisitForEachLoop(forEach, p).(*tree.ForEachLoop)
	v.insideLoop--
	return forEach
}

func (v *findChannelCreateInLoopVisitor) VisitMethodInvocation(mi *tree.MethodInvocation, p any) tree.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*tree.MethodInvocation)

	if v.insideLoop == 0 {
		return mi
	}

	// Must be a free-function call to "make" (no receiver).
	if mi.Select != nil || mi.Name.Name != "make" {
		return mi
	}

	// Check that the first real argument is a channel type.
	var realArgs []tree.Expression
	for _, arg := range mi.Arguments.Elements {
		if _, isEmpty := arg.Element.(*tree.Empty); !isEmpty {
			realArgs = append(realArgs, arg.Element)
		}
	}

	if len(realArgs) == 0 {
		return mi
	}

	if _, isChan := realArgs[0].(*tree.Channel); !isChan {
		return mi
	}

	mi = mi.WithMarkers(
		tree.FoundSearchResult(mi.Markers, "channel creation in loop; consider creating the channel once before the loop"),
	)
	return mi
}
