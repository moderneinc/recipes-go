/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package performance

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindDeferInLoop finds `defer` statements inside for/range loops.
// Deferred calls in loops accumulate and only execute when the enclosing
// function returns, which can lead to resource leaks and unexpected behavior.
type FindDeferInLoop struct {
	recipe.Base
}

func (r *FindDeferInLoop) Name() string {
	return "org.openrewrite.golang.codequality.FindDeferInLoop"
}
func (r *FindDeferInLoop) DisplayName() string { return "Find defer in loop" }
func (r *FindDeferInLoop) Description() string {
	return "Find `defer` statements inside for/range loops. Deferred calls accumulate and only run when the function exits, which can cause resource leaks."
}
func (r *FindDeferInLoop) Tags() []string { return []string{"performance"} }

func (r *FindDeferInLoop) Editor() recipe.TreeVisitor {
	return visitor.Init(&findDeferInLoopVisitor{})
}

type findDeferInLoopVisitor struct {
	visitor.GoVisitor
	insideLoop int
}

func (v *findDeferInLoopVisitor) VisitForLoop(forLoop *tree.ForLoop, p any) tree.J {
	v.insideLoop++
	forLoop = v.GoVisitor.VisitForLoop(forLoop, p).(*tree.ForLoop)
	v.insideLoop--
	return forLoop
}

func (v *findDeferInLoopVisitor) VisitForEachLoop(forEach *tree.ForEachLoop, p any) tree.J {
	v.insideLoop++
	forEach = v.GoVisitor.VisitForEachLoop(forEach, p).(*tree.ForEachLoop)
	v.insideLoop--
	return forEach
}

func (v *findDeferInLoopVisitor) VisitDefer(d *tree.Defer, p any) tree.J {
	d = v.GoVisitor.VisitDefer(d, p).(*tree.Defer)

	if v.insideLoop == 0 {
		return d
	}

	d = d.WithMarkers(
		tree.FoundSearchResult(d.Markers, "defer in loop; deferred calls accumulate until function exit"),
	)
	return d
}
