/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package redundancy

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// SimplifyGoroutineClosure simplifies `go func() { f() }()` patterns
// where a goroutine closure wraps a single function call, replacing them
// with `go f()`.
type SimplifyGoroutineClosure struct {
	recipe.Base
}

func (r *SimplifyGoroutineClosure) Name() string {
	return "org.openrewrite.golang.codequality.SimplifyGoroutineClosure"
}
func (r *SimplifyGoroutineClosure) DisplayName() string {
	return "Simplify goroutine closure"
}
func (r *SimplifyGoroutineClosure) Description() string {
	return "Simplify `go func() { f() }()` to `go f()` where the closure wraps a single function call."
}
func (r *SimplifyGoroutineClosure) Tags() []string {
	return []string{"cleanup", "redundancy"}
}

func (r *SimplifyGoroutineClosure) Editor() recipe.TreeVisitor {
	return visitor.Init(&simplifyGoroutineClosureVisitor{})
}

type simplifyGoroutineClosureVisitor struct {
	visitor.GoVisitor
}

func (v *simplifyGoroutineClosureVisitor) VisitGoStmt(g *tree.GoStmt, p any) tree.J {
	g = v.GoVisitor.VisitGoStmt(g, p).(*tree.GoStmt)

	// The expression must be a function call (MethodInvocation).
	mi, ok := g.Expr.(*tree.MethodInvocation)
	if !ok {
		return g
	}

	// The call's Select must be a function literal (MethodDeclaration),
	// possibly wrapped in StatementExpression.
	if mi.Select == nil {
		return g
	}
	var funcLit *tree.MethodDeclaration
	switch sel := mi.Select.Element.(type) {
	case *tree.MethodDeclaration:
		funcLit = sel
	case *tree.StatementExpression:
		if md, ok := sel.Statement.(*tree.MethodDeclaration); ok {
			funcLit = md
		}
	}
	if funcLit == nil {
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
	innerCall, isCall := realStmts[0].(*tree.MethodInvocation)
	if !isCall {
		return g
	}

	// Replace the closure call with the inner call, preserving the go statement's prefix.
	// Set the inner call's prefix to a single space (the space between "go" and the call)
	// and ensure the Name prefix is empty to avoid double spacing.
	replaced := innerCall.WithPrefix(tree.SingleSpace)
	replaced = replaced.WithName(replaced.Name.WithPrefix(tree.EmptySpace))
	return &tree.GoStmt{
		ID:      g.ID,
		Prefix:  g.Prefix,
		Markers: g.Markers,
		Expr:    replaced,
	}
}
