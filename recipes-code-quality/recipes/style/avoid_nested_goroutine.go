/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// AvoidNestedGoroutine finds goroutines launched inside other goroutines.
// Nested goroutines create hard-to-track concurrency and make it difficult
// to reason about program flow and resource lifetimes.
type AvoidNestedGoroutine struct {
	recipe.Base
}

func (r *AvoidNestedGoroutine) Name() string {
	return "org.openrewrite.golang.codequality.AvoidNestedGoroutine"
}
func (r *AvoidNestedGoroutine) DisplayName() string { return "Avoid nested goroutine" }
func (r *AvoidNestedGoroutine) Description() string {
	return "Find goroutines launched inside other goroutines. Nested goroutines create hard-to-track concurrency."
}
func (r *AvoidNestedGoroutine) Tags() []string { return []string{"style", "concurrency"} }

func (r *AvoidNestedGoroutine) Editor() recipe.TreeVisitor {
	return visitor.Init(&avoidNestedGoroutineVisitor{})
}

type avoidNestedGoroutineVisitor struct {
	visitor.GoVisitor
	goDepth int
}

func (v *avoidNestedGoroutineVisitor) VisitGoStmt(g *golang.GoStmt, p any) java.J {
	if v.goDepth > 0 {
		g = g.WithMarkers(
			java.MarkupWarn(g.Markers, "nested goroutine; consider restructuring to avoid goroutines inside goroutines"),
		)
	}

	v.goDepth++
	g = v.GoVisitor.VisitGoStmt(g, p).(*golang.GoStmt)
	v.goDepth--

	return g
}
