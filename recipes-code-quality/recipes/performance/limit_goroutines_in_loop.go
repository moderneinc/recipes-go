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

// LimitGoroutinesInLoop finds `go` statements inside for/range loops.
// Launching goroutines in a loop without bounds can lead to unbounded
// goroutine creation and resource exhaustion.
type LimitGoroutinesInLoop struct {
	recipe.Base
}

func (r *LimitGoroutinesInLoop) Name() string {
	return "org.openrewrite.golang.codequality.LimitGoroutinesInLoop"
}
func (r *LimitGoroutinesInLoop) DisplayName() string { return "Limit goroutines in loop" }
func (r *LimitGoroutinesInLoop) Description() string {
	return "Find `go` statements inside for/range loops. Unbounded goroutine creation can cause resource exhaustion; consider using a worker pool."
}
func (r *LimitGoroutinesInLoop) Tags() []string { return []string{"performance"} }

func (r *LimitGoroutinesInLoop) Editor() recipe.TreeVisitor {
	return visitor.Init(&limitGoroutinesInLoopVisitor{})
}

type limitGoroutinesInLoopVisitor struct {
	visitor.GoVisitor
	insideLoop int
}

func (v *limitGoroutinesInLoopVisitor) VisitForLoop(forLoop *java.ForLoop, p any) java.J {
	v.insideLoop++
	forLoop = v.GoVisitor.VisitForLoop(forLoop, p).(*java.ForLoop)
	v.insideLoop--
	return forLoop
}

func (v *limitGoroutinesInLoopVisitor) VisitForEachLoop(forEach *java.ForEachLoop, p any) java.J {
	v.insideLoop++
	forEach = v.GoVisitor.VisitForEachLoop(forEach, p).(*java.ForEachLoop)
	v.insideLoop--
	return forEach
}

func (v *limitGoroutinesInLoopVisitor) VisitGoStmt(g *golang.GoStmt, p any) java.J {
	g = v.GoVisitor.VisitGoStmt(g, p).(*golang.GoStmt)

	if v.insideLoop == 0 {
		return g
	}

	g = g.WithMarkers(
		java.MarkupWarn(g.Markers, "goroutine launched in loop; unbounded goroutine creation can cause resource exhaustion"),
	)
	return g
}
