/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package performance

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindNewInLoop finds calls to the built-in `new()` function inside for/range
// loops. Allocating with `new()` on every iteration adds GC pressure; consider
// reusing the object or allocating once before the loop.
type FindNewInLoop struct {
	recipe.Base
}

func (r *FindNewInLoop) Name() string {
	return "org.openrewrite.golang.codequality.FindNewInLoop"
}
func (r *FindNewInLoop) DisplayName() string { return "Find new() allocation in loop" }
func (r *FindNewInLoop) Description() string {
	return "Find `new()` calls inside for/range loops. Repeated heap allocations in loops add GC pressure; consider reusing the object."
}
func (r *FindNewInLoop) Tags() []string { return []string{"performance"} }

func (r *FindNewInLoop) Editor() recipe.TreeVisitor {
	return visitor.Init(&findNewInLoopVisitor{})
}

type findNewInLoopVisitor struct {
	visitor.GoVisitor
	insideLoop int
}

func (v *findNewInLoopVisitor) VisitForLoop(forLoop *tree.ForLoop, p any) tree.J {
	v.insideLoop++
	forLoop = v.GoVisitor.VisitForLoop(forLoop, p).(*tree.ForLoop)
	v.insideLoop--
	return forLoop
}

func (v *findNewInLoopVisitor) VisitForEachLoop(forEach *tree.ForEachLoop, p any) tree.J {
	v.insideLoop++
	forEach = v.GoVisitor.VisitForEachLoop(forEach, p).(*tree.ForEachLoop)
	v.insideLoop--
	return forEach
}

func (v *findNewInLoopVisitor) VisitMethodInvocation(mi *tree.MethodInvocation, p any) tree.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*tree.MethodInvocation)

	if v.insideLoop == 0 {
		return mi
	}

	// Match: new(...) — built-in, so no Select and Name == "new".
	if mi.Select != nil || mi.Name.Name != "new" {
		return mi
	}

	mi = mi.WithMarkers(
		tree.FoundSearchResult(mi.Markers, "new() in loop; consider allocating once outside the loop"),
	)
	return mi
}
