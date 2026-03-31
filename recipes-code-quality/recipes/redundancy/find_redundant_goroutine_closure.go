/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package redundancy

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindRedundantGoroutineClosure finds `go func() { f() }()` patterns where a
// goroutine closure wraps a single function call. This is redundant and can be
// simplified to `go f()`.
type FindRedundantGoroutineClosure struct {
	recipe.Base
}

func (r *FindRedundantGoroutineClosure) Name() string {
	return "org.openrewrite.golang.codequality.FindRedundantGoroutineClosure"
}
func (r *FindRedundantGoroutineClosure) DisplayName() string {
	return "Find redundant goroutine closure"
}
func (r *FindRedundantGoroutineClosure) Description() string {
	return "Find `go func() { f() }()` where the closure wraps a single function call. Can be simplified to `go f()`."
}
func (r *FindRedundantGoroutineClosure) Tags() []string {
	return []string{"cleanup", "redundancy"}
}

func (r *FindRedundantGoroutineClosure) Editor() recipe.TreeVisitor {
	return visitor.Init(&findRedundantGoroutineClosureVisitor{})
}

type findRedundantGoroutineClosureVisitor struct {
	visitor.GoVisitor
}

func (v *findRedundantGoroutineClosureVisitor) VisitGoStmt(g *tree.GoStmt, p any) tree.J {
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

	// The function literal must have a body with exactly 1 real statement.
	if funcLit.Body == nil {
		return g
	}
	var realStmts []tree.Statement
	for _, stmt := range funcLit.Body.Statements {
		if _, isEmpty := stmt.Element.(*tree.Empty); !isEmpty {
			realStmts = append(realStmts, stmt.Element)
		}
	}
	if len(realStmts) != 1 {
		return g
	}

	// That single statement must be a MethodInvocation (a function call).
	if _, isCall := realStmts[0].(*tree.MethodInvocation); !isCall {
		return g
	}

	g = g.WithMarkers(
		tree.FoundSearchResult(g.Markers, "redundant goroutine closure wrapping a single function call"),
	)
	return g
}
