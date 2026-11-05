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

// RemoveEmptyGoroutine removes `go func() {}()` patterns where the goroutine
// body is empty. An empty goroutine does nothing and is dead code.
type RemoveEmptyGoroutine struct {
	recipe.Base
}

func (r *RemoveEmptyGoroutine) Name() string {
	return "org.openrewrite.golang.codequality.RemoveEmptyGoroutine"
}
func (r *RemoveEmptyGoroutine) DisplayName() string { return "Remove empty goroutine" }
func (r *RemoveEmptyGoroutine) Description() string {
	return "Remove `go func() {}()` patterns where the goroutine body is empty."
}
func (r *RemoveEmptyGoroutine) Tags() []string { return []string{"concurrency"} }

func (r *RemoveEmptyGoroutine) Editor() recipe.TreeVisitor {
	return visitor.Init(&removeEmptyGoroutineVisitor{})
}

type removeEmptyGoroutineVisitor struct {
	visitor.GoVisitor
}

func (v *removeEmptyGoroutineVisitor) VisitGoStmt(g *golang.GoStmt, p any) java.J {
	g = v.GoVisitor.VisitGoStmt(g, p).(*golang.GoStmt)

	// The expression must be a function call (MethodInvocation).
	mi, ok := g.Expr.(*java.MethodInvocation)
	if !ok {
		return g
	}

	if mi.Select == nil {
		return g
	}
	sel, ok := mi.Select.Element.(*golang.StatementExpression)
	if !ok {
		return g
	}
	funcLit, ok := sel.Statement.(*java.MethodDeclaration)
	if !ok {
		return g
	}

	// The function literal must have an empty body.
	if funcLit.Body == nil {
		return g
	}

	// Check that the body has no real statements (only Empty sentinels).
	for _, stmt := range funcLit.Body.Statements {
		if _, isEmpty := stmt.Element.(*java.Empty); !isEmpty {
			return g
		}
	}

	// Remove the empty goroutine.
	return &java.Empty{}
}
