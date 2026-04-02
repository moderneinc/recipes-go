/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// AuditJsonNumber finds usage of `json.Number`. The json.Number type should
// be used carefully as it can lead to unexpected behavior when converting
// between numeric types.
type AuditJsonNumber struct {
	recipe.Base
}

func (r *AuditJsonNumber) Name() string {
	return "org.openrewrite.golang.codequality.AuditJsonNumber"
}
func (r *AuditJsonNumber) DisplayName() string { return "Audit json.Number" }
func (r *AuditJsonNumber) Description() string {
	return "Find usage of `json.Number`. json.Number should be used carefully as it can lead to unexpected behavior when converting between numeric types."
}
func (r *AuditJsonNumber) Tags() []string { return []string{"style"} }

func (r *AuditJsonNumber) Editor() recipe.TreeVisitor {
	return visitor.Init(&auditJsonNumberVisitor{})
}

type auditJsonNumberVisitor struct {
	visitor.GoVisitor
}

func (v *auditJsonNumberVisitor) VisitFieldAccess(fa *tree.FieldAccess, p any) tree.J {
	fa = v.GoVisitor.VisitFieldAccess(fa, p).(*tree.FieldAccess)

	ident, ok := fa.Target.(*tree.Identifier)
	if !ok || ident.Name != "json" {
		return fa
	}

	if fa.Name.Element.Name != "Number" {
		return fa
	}

	fa = fa.WithMarkers(tree.MarkupInfo(fa.Markers, "json.Number should be used carefully"))
	return fa
}
