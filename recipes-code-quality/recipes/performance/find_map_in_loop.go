/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package performance

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindMapAllocInLoop finds `make(map[...]...)` calls inside for/range loops.
// Allocating a new map on every iteration is wasteful when the map could be
// allocated once before the loop and cleared with `clear()` or `delete` each iteration.
type FindMapAllocInLoop struct {
	recipe.Base
}

func (r *FindMapAllocInLoop) Name() string {
	return "org.openrewrite.golang.codequality.FindMapAllocInLoop"
}
func (r *FindMapAllocInLoop) DisplayName() string { return "Find map allocation in loop" }
func (r *FindMapAllocInLoop) Description() string {
	return "Find `make(map[...]...)` calls inside for/range loops. Consider allocating the map once and clearing it each iteration."
}
func (r *FindMapAllocInLoop) Tags() []string { return []string{"performance"} }

func (r *FindMapAllocInLoop) Editor() recipe.TreeVisitor {
	return visitor.Init(&findMapAllocInLoopVisitor{})
}

type findMapAllocInLoopVisitor struct {
	visitor.GoVisitor
	insideLoop int
}

func (v *findMapAllocInLoopVisitor) VisitForLoop(forLoop *tree.ForLoop, p any) tree.J {
	v.insideLoop++
	forLoop = v.GoVisitor.VisitForLoop(forLoop, p).(*tree.ForLoop)
	v.insideLoop--
	return forLoop
}

func (v *findMapAllocInLoopVisitor) VisitForEachLoop(forEach *tree.ForEachLoop, p any) tree.J {
	v.insideLoop++
	forEach = v.GoVisitor.VisitForEachLoop(forEach, p).(*tree.ForEachLoop)
	v.insideLoop--
	return forEach
}

func (v *findMapAllocInLoopVisitor) VisitMethodInvocation(mi *tree.MethodInvocation, p any) tree.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*tree.MethodInvocation)

	if v.insideLoop == 0 {
		return mi
	}

	// Must be a free-function call to "make" (no receiver).
	if mi.Select != nil || mi.Name.Name != "make" {
		return mi
	}

	// Check that the first real argument is a map type.
	var realArgs []tree.Expression
	for _, arg := range mi.Arguments.Elements {
		if _, isEmpty := arg.Element.(*tree.Empty); !isEmpty {
			realArgs = append(realArgs, arg.Element)
		}
	}

	if len(realArgs) == 0 {
		return mi
	}

	if _, isMap := realArgs[0].(*tree.MapType); !isMap {
		return mi
	}

	mi = mi.WithMarkers(
		tree.FoundSearchResult(mi.Markers, "map allocation in loop; consider allocating once and clearing"),
	)
	return mi
}
