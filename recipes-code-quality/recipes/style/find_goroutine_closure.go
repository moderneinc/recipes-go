/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindGoroutineClosure finds `go func() { ... }()` patterns where a goroutine
// is launched with an anonymous function literal. These closures can
// inadvertently capture loop variables, leading to subtle concurrency bugs.
type FindGoroutineClosure struct {
	recipe.Base
}

func (r *FindGoroutineClosure) Name() string {
	return "org.openrewrite.golang.codequality.FindGoroutineClosure"
}
func (r *FindGoroutineClosure) DisplayName() string { return "Find goroutine closures" }
func (r *FindGoroutineClosure) Description() string {
	return "Find `go func() { ... }()` patterns. Goroutines with closures can inadvertently capture loop variables."
}
func (r *FindGoroutineClosure) Tags() []string { return []string{"concurrency"} }

func (r *FindGoroutineClosure) Editor() recipe.TreeVisitor {
	return visitor.Init(&findGoroutineClosureVisitor{})
}

type findGoroutineClosureVisitor struct {
	visitor.GoVisitor
}

func (v *findGoroutineClosureVisitor) VisitGoStmt(g *tree.GoStmt, p any) tree.J {
	g = v.GoVisitor.VisitGoStmt(g, p).(*tree.GoStmt)

	// The expression must be a function call (MethodInvocation).
	mi, ok := g.Expr.(*tree.MethodInvocation)
	if !ok {
		return g
	}

	// The call's Select must be a function literal (MethodDeclaration).
	if mi.Select == nil {
		return g
	}
	if _, isFuncLit := mi.Select.Element.(*tree.MethodDeclaration); !isFuncLit {
		return g
	}

	g = g.WithMarkers(tree.FoundSearchResult(g.Markers, "goroutine with closure"))
	return g
}
