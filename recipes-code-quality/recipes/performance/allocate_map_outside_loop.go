/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package performance

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// AllocateMapOutsideLoop finds `make(map[...]...)` calls inside for/range loops.
// Allocating a new map on every iteration is wasteful when the map could be
// allocated once before the loop and cleared with `clear()` or `delete` each iteration.
type AllocateMapOutsideLoop struct {
	recipe.Base
}

func (r *AllocateMapOutsideLoop) Name() string {
	return "org.openrewrite.golang.codequality.AllocateMapOutsideLoop"
}
func (r *AllocateMapOutsideLoop) DisplayName() string { return "Allocate map outside loop" }
func (r *AllocateMapOutsideLoop) Description() string {
	return "Find `make(map[...]...)` calls inside for/range loops. Consider allocating the map once and clearing it each iteration."
}
func (r *AllocateMapOutsideLoop) Tags() []string { return []string{"performance"} }

func (r *AllocateMapOutsideLoop) Editor() recipe.TreeVisitor {
	return visitor.Init(&allocateMapOutsideLoopVisitor{})
}

type allocateMapOutsideLoopVisitor struct {
	visitor.GoVisitor
	insideLoop int
}

func (v *allocateMapOutsideLoopVisitor) VisitForLoop(forLoop *tree.ForLoop, p any) tree.J {
	v.insideLoop++
	forLoop = v.GoVisitor.VisitForLoop(forLoop, p).(*tree.ForLoop)
	v.insideLoop--
	return forLoop
}

func (v *allocateMapOutsideLoopVisitor) VisitForEachLoop(forEach *tree.ForEachLoop, p any) tree.J {
	v.insideLoop++
	forEach = v.GoVisitor.VisitForEachLoop(forEach, p).(*tree.ForEachLoop)
	v.insideLoop--
	return forEach
}

func (v *allocateMapOutsideLoopVisitor) VisitMethodInvocation(mi *tree.MethodInvocation, p any) tree.J {
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
		tree.MarkupInfo(mi.Markers, "map allocation in loop; consider allocating once and clearing"),
	)
	return mi
}
