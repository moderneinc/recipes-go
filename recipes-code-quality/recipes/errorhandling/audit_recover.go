/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// AuditRecover finds calls to the built-in `recover()` function. Recover
// catches panics and should only appear inside deferred functions. Its
// presence often signals unusual control flow that deserves review.
type AuditRecover struct {
	recipe.Base
}

func (r *AuditRecover) Name() string {
	return "org.openrewrite.golang.codequality.AuditRecover"
}
func (r *AuditRecover) DisplayName() string { return "Audit recover() calls" }
func (r *AuditRecover) Description() string {
	return "Find calls to the built-in `recover()` function, which catches panics and signals unusual control flow."
}
func (r *AuditRecover) Tags() []string { return []string{"error-handling"} }

func (r *AuditRecover) Editor() recipe.TreeVisitor {
	return visitor.Init(&auditRecoverVisitor{})
}

type auditRecoverVisitor struct {
	visitor.GoVisitor
}

func (v *auditRecoverVisitor) VisitMethodInvocation(mi *tree.MethodInvocation, p any) tree.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*tree.MethodInvocation)

	// Match: recover() — built-in, so no Select and Name == "recover".
	if mi.Select != nil || mi.Name.Name != "recover" {
		return mi
	}

	mi = mi.WithMarkers(
		tree.MarkupInfo(mi.Markers, "recover() catches panics; ensure it is in a deferred function"),
	)
	return mi
}
