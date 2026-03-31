/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindEmptyGoroutine finds `go func() {}()` patterns where the goroutine body
// is empty. An empty goroutine does nothing and is likely dead code or an
// incomplete implementation.
type FindEmptyGoroutine struct {
	recipe.Base
}

func (r *FindEmptyGoroutine) Name() string {
	return "org.openrewrite.golang.codequality.FindEmptyGoroutine"
}
func (r *FindEmptyGoroutine) DisplayName() string { return "Find empty goroutines" }
func (r *FindEmptyGoroutine) Description() string {
	return "Find `go func() {}()` patterns where the goroutine body is empty."
}
func (r *FindEmptyGoroutine) Tags() []string { return []string{"concurrency"} }

func (r *FindEmptyGoroutine) Editor() recipe.TreeVisitor {
	return visitor.Init(&findEmptyGoroutineVisitor{})
}

type findEmptyGoroutineVisitor struct {
	visitor.GoVisitor
}

func (v *findEmptyGoroutineVisitor) VisitGoStmt(g *tree.GoStmt, p any) tree.J {
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
	funcLit, isFuncLit := mi.Select.Element.(*tree.MethodDeclaration)
	if !isFuncLit {
		return g
	}

	// The function literal must have an empty body.
	if funcLit.Body == nil {
		return g
	}

	// Check that the body has no real statements (only Empty sentinels).
	for _, stmt := range funcLit.Body.Statements {
		if _, isEmpty := stmt.Element.(*tree.Empty); !isEmpty {
			return g
		}
	}

	g = g.WithMarkers(tree.FoundSearchResult(g.Markers, "empty goroutine"))
	return g
}
