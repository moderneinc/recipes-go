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

// AuditGoroutineClosure finds `go func() { ... }()` patterns where a goroutine
// is launched with an anonymous function literal. These closures can
// inadvertently capture loop variables, leading to subtle concurrency bugs.
type AuditGoroutineClosure struct {
	recipe.Base
}

func (r *AuditGoroutineClosure) Name() string {
	return "org.openrewrite.golang.codequality.AuditGoroutineClosure"
}
func (r *AuditGoroutineClosure) DisplayName() string { return "Audit goroutine closure" }
func (r *AuditGoroutineClosure) Description() string {
	return "Find `go func() { ... }()` patterns. Goroutines with closures can inadvertently capture loop variables."
}
func (r *AuditGoroutineClosure) Tags() []string { return []string{"concurrency"} }

func (r *AuditGoroutineClosure) Editor() recipe.TreeVisitor {
	return visitor.Init(&auditGoroutineClosureVisitor{})
}

type auditGoroutineClosureVisitor struct {
	visitor.GoVisitor
}

func (v *auditGoroutineClosureVisitor) VisitGoStmt(g *golang.GoStmt, p any) java.J {
	g = v.GoVisitor.VisitGoStmt(g, p).(*golang.GoStmt)

	// The expression must be a function call (MethodInvocation).
	mi, ok := g.Expr.(*java.MethodInvocation)
	if !ok {
		return g
	}

	// The call's Select must be a function literal (MethodDeclaration),
	// possibly wrapped in StatementExpression.
	if mi.Select == nil {
		return g
	}
	isFuncLit := false
	switch sel := mi.Select.Element.(type) {
	case *java.MethodDeclaration:
		isFuncLit = true
	case *golang.StatementExpression:
		_, isFuncLit = sel.Statement.(*java.MethodDeclaration)
	}
	if !isFuncLit {
		return g
	}

	g = g.WithMarkers(java.MarkupInfo(g.Markers, "goroutine with closure"))
	return g
}
