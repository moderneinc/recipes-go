/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree"
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

func (v *avoidNestedGoroutineVisitor) VisitGoStmt(g *tree.GoStmt, p any) tree.J {
	g = v.GoVisitor.VisitGoStmt(g, p).(*tree.GoStmt)

	if v.goDepth > 0 {
		g = g.WithMarkers(
			tree.MarkupWarn(g.Markers, "nested goroutine; consider restructuring to avoid goroutines inside goroutines"),
		)
	}

	return g
}

func (v *avoidNestedGoroutineVisitor) VisitMethodDeclaration(md *tree.MethodDeclaration, p any) tree.J {
	// Track goroutine depth: if the MethodDeclaration is a function literal
	// called via go, the GoStmt has already been visited and goDepth incremented.
	// We detect this by checking if our parent context is inside a GoStmt.
	// Instead, we increment goDepth around the body of a GoStmt's function literal.
	md = v.GoVisitor.VisitMethodDeclaration(md, p).(*tree.MethodDeclaration)
	return md
}

func (v *avoidNestedGoroutineVisitor) VisitBlock(block *tree.Block, p any) tree.J {
	block = v.GoVisitor.VisitBlock(block, p).(*tree.Block)
	return block
}

// Visit overrides the default to track goroutine depth properly.
// When we encounter a GoStmt, we increment depth before visiting its Expr
// (which the default VisitGoStmt does not recurse into, but the block visitor
// will handle the func literal body through VisitMethodDeclaration).
func (v *avoidNestedGoroutineVisitor) Visit(t tree.Tree, p any) tree.Tree {
	if g, ok := t.(*tree.GoStmt); ok {
		// First, call VisitGoStmt which marks if nested
		result := v.GoVisitor.Self.(visitor.VisitorI).VisitGoStmt(g, p)

		// Then increment depth and visit the expression (func literal + call)
		v.goDepth++
		resultG := result.(*tree.GoStmt)
		resultG.Expr = v.GoVisitor.Self.(visitor.VisitorI).Visit(resultG.Expr, p).(tree.Expression)
		v.goDepth--

		return resultG
	}

	return v.GoVisitor.Visit(t, p)
}
