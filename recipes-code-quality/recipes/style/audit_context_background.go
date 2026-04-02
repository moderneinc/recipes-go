/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// AuditContextBackground finds calls to `context.Background()`. In most
// application code the context should be propagated from the caller rather
// than creating a new empty context via `context.Background()`.
type AuditContextBackground struct {
	recipe.Base
}

func (r *AuditContextBackground) Name() string {
	return "org.openrewrite.golang.codequality.AuditContextBackground"
}
func (r *AuditContextBackground) DisplayName() string { return "Audit context.Background" }
func (r *AuditContextBackground) Description() string {
	return "Find calls to `context.Background()`. Consider using a context passed from the caller instead."
}
func (r *AuditContextBackground) Tags() []string { return []string{"style"} }

func (r *AuditContextBackground) Editor() recipe.TreeVisitor {
	return visitor.Init(&auditContextBackgroundVisitor{})
}

type auditContextBackgroundVisitor struct {
	visitor.GoVisitor
}

func (v *auditContextBackgroundVisitor) VisitMethodInvocation(mi *tree.MethodInvocation, p any) tree.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*tree.MethodInvocation)

	if mi.Select == nil {
		return mi
	}

	ident, ok := mi.Select.Element.(*tree.Identifier)
	if !ok || ident.Name != "context" {
		return mi
	}

	if mi.Name.Name != "Background" {
		return mi
	}

	mi = mi.WithMarkers(tree.MarkupInfo(mi.Markers, "context.Background() call; consider using a passed context instead"))
	return mi
}
