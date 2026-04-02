/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree"
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

func (v *auditGoroutineClosureVisitor) VisitGoStmt(g *tree.GoStmt, p any) tree.J {
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

	g = g.WithMarkers(tree.MarkupInfo(g.Markers, "goroutine with closure"))
	return g
}
