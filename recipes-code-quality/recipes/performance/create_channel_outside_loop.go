/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package performance

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// CreateChannelOutsideLoop finds `make(chan ...)` calls inside for/range loops.
// Creating a channel on every iteration is usually unnecessary; the channel
// should be created once before the loop.
type CreateChannelOutsideLoop struct {
	recipe.Base
}

func (r *CreateChannelOutsideLoop) Name() string {
	return "org.openrewrite.golang.codequality.CreateChannelOutsideLoop"
}
func (r *CreateChannelOutsideLoop) DisplayName() string { return "Create channel outside loop" }
func (r *CreateChannelOutsideLoop) Description() string {
	return "Find `make(chan ...)` calls inside for/range loops. Channel creation in loops suggests the channel should be created once before the loop."
}
func (r *CreateChannelOutsideLoop) Tags() []string { return []string{"performance"} }

func (r *CreateChannelOutsideLoop) Editor() recipe.TreeVisitor {
	return visitor.Init(&createChannelOutsideLoopVisitor{})
}

type createChannelOutsideLoopVisitor struct {
	visitor.GoVisitor
	insideLoop int
}

func (v *createChannelOutsideLoopVisitor) VisitForLoop(forLoop *java.ForLoop, p any) java.J {
	v.insideLoop++
	forLoop = v.GoVisitor.VisitForLoop(forLoop, p).(*java.ForLoop)
	v.insideLoop--
	return forLoop
}

func (v *createChannelOutsideLoopVisitor) VisitForEachLoop(forEach *java.ForEachLoop, p any) java.J {
	v.insideLoop++
	forEach = v.GoVisitor.VisitForEachLoop(forEach, p).(*java.ForEachLoop)
	v.insideLoop--
	return forEach
}

func (v *createChannelOutsideLoopVisitor) VisitMethodInvocation(mi *java.MethodInvocation, p any) java.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*java.MethodInvocation)

	if v.insideLoop == 0 {
		return mi
	}

	// Must be a free-function call to "make" (no receiver).
	if mi.Select != nil || mi.Name.Name != "make" {
		return mi
	}

	// Check that the first real argument is a channel type.
	var realArgs []java.Expression
	for _, arg := range mi.Arguments.Elements {
		if _, isEmpty := arg.Element.(*java.Empty); !isEmpty {
			realArgs = append(realArgs, arg.Element)
		}
	}

	if len(realArgs) == 0 {
		return mi
	}

	if _, isChan := realArgs[0].(*golang.Channel); !isChan {
		return mi
	}

	mi = mi.WithMarkers(
		java.MarkupInfo(mi.Markers, "channel creation in loop; consider creating the channel once before the loop"),
	)
	return mi
}
