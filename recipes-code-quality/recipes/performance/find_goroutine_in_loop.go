/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package performance

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindGoroutineInLoop finds `go` statements inside for/range loops.
// Launching goroutines in a loop without bounds can lead to unbounded
// goroutine creation and resource exhaustion.
type FindGoroutineInLoop struct {
	recipe.Base
}

func (r *FindGoroutineInLoop) Name() string {
	return "org.openrewrite.golang.codequality.FindGoroutineInLoop"
}
func (r *FindGoroutineInLoop) DisplayName() string { return "Find goroutine launch in loop" }
func (r *FindGoroutineInLoop) Description() string {
	return "Find `go` statements inside for/range loops. Unbounded goroutine creation can cause resource exhaustion; consider using a worker pool."
}
func (r *FindGoroutineInLoop) Tags() []string { return []string{"performance"} }

func (r *FindGoroutineInLoop) Editor() recipe.TreeVisitor {
	return visitor.Init(&findGoroutineInLoopVisitor{})
}

type findGoroutineInLoopVisitor struct {
	visitor.GoVisitor
	insideLoop int
}

func (v *findGoroutineInLoopVisitor) VisitForLoop(forLoop *tree.ForLoop, p any) tree.J {
	v.insideLoop++
	forLoop = v.GoVisitor.VisitForLoop(forLoop, p).(*tree.ForLoop)
	v.insideLoop--
	return forLoop
}

func (v *findGoroutineInLoopVisitor) VisitForEachLoop(forEach *tree.ForEachLoop, p any) tree.J {
	v.insideLoop++
	forEach = v.GoVisitor.VisitForEachLoop(forEach, p).(*tree.ForEachLoop)
	v.insideLoop--
	return forEach
}

func (v *findGoroutineInLoopVisitor) VisitGoStmt(g *tree.GoStmt, p any) tree.J {
	g = v.GoVisitor.VisitGoStmt(g, p).(*tree.GoStmt)

	if v.insideLoop == 0 {
		return g
	}

	g = g.WithMarkers(
		tree.FoundSearchResult(g.Markers, "goroutine launched in loop; unbounded goroutine creation can cause resource exhaustion"),
	)
	return g
}
