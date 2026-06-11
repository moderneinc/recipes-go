/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// AuditJsonRawMessage finds usage of `json.RawMessage`. RawMessage defers
// JSON parsing and should be reviewed to ensure the deferred parsing is
// handled correctly.
type AuditJsonRawMessage struct {
	recipe.Base
}

func (r *AuditJsonRawMessage) Name() string {
	return "org.openrewrite.golang.codequality.AuditJsonRawMessage"
}
func (r *AuditJsonRawMessage) DisplayName() string { return "Audit json.RawMessage" }
func (r *AuditJsonRawMessage) Description() string {
	return "Find usage of `json.RawMessage`. RawMessage defers JSON parsing and should be reviewed to ensure deferred parsing is handled correctly."
}
func (r *AuditJsonRawMessage) Tags() []string { return []string{"style"} }

func (r *AuditJsonRawMessage) Editor() recipe.TreeVisitor {
	return visitor.Init(&auditJsonRawMessageVisitor{})
}

type auditJsonRawMessageVisitor struct {
	visitor.GoVisitor
}

func (v *auditJsonRawMessageVisitor) VisitFieldAccess(fa *java.FieldAccess, p any) java.J {
	fa = v.GoVisitor.VisitFieldAccess(fa, p).(*java.FieldAccess)

	ident, ok := fa.Target.(*java.Identifier)
	if !ok || ident.Name != "json" {
		return fa
	}

	if fa.Name.Element.Name != "RawMessage" {
		return fa
	}

	fa = fa.WithMarkers(java.MarkupInfo(fa.Markers, "json.RawMessage defers parsing; review for correctness"))
	return fa
}
