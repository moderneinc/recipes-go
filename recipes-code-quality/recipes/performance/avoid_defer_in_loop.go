/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package performance

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// AvoidDeferInLoop finds `defer` statements inside for/range loops.
// Deferred calls in loops accumulate and only execute when the enclosing
// function returns, which can lead to resource leaks and unexpected behavior.
type AvoidDeferInLoop struct {
	recipe.Base
}

func (r *AvoidDeferInLoop) Name() string {
	return "org.openrewrite.golang.codequality.AvoidDeferInLoop"
}
func (r *AvoidDeferInLoop) DisplayName() string { return "Avoid defer in loop" }
func (r *AvoidDeferInLoop) Description() string {
	return "Find `defer` statements inside for/range loops. Deferred calls accumulate and only run when the function exits, which can cause resource leaks."
}
func (r *AvoidDeferInLoop) Tags() []string { return []string{"performance"} }

func (r *AvoidDeferInLoop) Editor() recipe.TreeVisitor {
	return visitor.Init(&avoidDeferInLoopVisitor{})
}

type avoidDeferInLoopVisitor struct {
	visitor.GoVisitor
	insideLoop int
}

func (v *avoidDeferInLoopVisitor) VisitForLoop(forLoop *tree.ForLoop, p any) tree.J {
	v.insideLoop++
	forLoop = v.GoVisitor.VisitForLoop(forLoop, p).(*tree.ForLoop)
	v.insideLoop--
	return forLoop
}

func (v *avoidDeferInLoopVisitor) VisitForEachLoop(forEach *tree.ForEachLoop, p any) tree.J {
	v.insideLoop++
	forEach = v.GoVisitor.VisitForEachLoop(forEach, p).(*tree.ForEachLoop)
	v.insideLoop--
	return forEach
}

func (v *avoidDeferInLoopVisitor) VisitDefer(d *tree.Defer, p any) tree.J {
	d = v.GoVisitor.VisitDefer(d, p).(*tree.Defer)

	if v.insideLoop == 0 {
		return d
	}

	d = d.WithMarkers(
		tree.MarkupWarn(d.Markers, "defer in loop; deferred calls accumulate until function exit"),
	)
	return d
}
