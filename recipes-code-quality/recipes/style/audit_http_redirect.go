/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// AuditHttpRedirect finds calls to `http.Redirect`. These calls should be
// reviewed to ensure redirect targets are validated and status codes are
// appropriate.
type AuditHttpRedirect struct {
	recipe.Base
}

func (r *AuditHttpRedirect) Name() string {
	return "org.openrewrite.golang.codequality.AuditHttpRedirect"
}
func (r *AuditHttpRedirect) DisplayName() string { return "Audit HTTP redirect" }
func (r *AuditHttpRedirect) Description() string {
	return "Find calls to `http.Redirect`. Review redirect targets to ensure they are validated and status codes are appropriate."
}
func (r *AuditHttpRedirect) Tags() []string { return []string{"style", "net/http"} }

func (r *AuditHttpRedirect) Editor() recipe.TreeVisitor {
	return visitor.Init(&auditHttpRedirectVisitor{})
}

type auditHttpRedirectVisitor struct {
	visitor.GoVisitor
}

func (v *auditHttpRedirectVisitor) VisitMethodInvocation(mi *java.MethodInvocation, p any) java.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*java.MethodInvocation)

	if mi.Select == nil {
		return mi
	}

	ident, ok := mi.Select.Element.(*java.Identifier)
	if !ok || ident.Name != "http" {
		return mi
	}

	if mi.Name.Name != "Redirect" {
		return mi
	}

	mi = mi.WithMarkers(java.MarkupInfo(mi.Markers, "review redirect target and status code"))
	return mi
}
