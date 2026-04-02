/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling

import (
	"strings"

	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// AuditMustFunction finds calls to functions named `Must*` or `must*`. These
// typically panic on error and should be used carefully — generally only in
// package-level variable initialization, not in request-handling code.
type AuditMustFunction struct {
	recipe.Base
}

func (r *AuditMustFunction) Name() string {
	return "org.openrewrite.golang.codequality.AuditMustFunction"
}
func (r *AuditMustFunction) DisplayName() string { return "Audit Must* function calls" }
func (r *AuditMustFunction) Description() string {
	return "Find calls to functions named `Must*` or `must*`, which typically panic on error."
}
func (r *AuditMustFunction) Tags() []string { return []string{"error-handling", "lint"} }

func (r *AuditMustFunction) Editor() recipe.TreeVisitor {
	return visitor.Init(&auditMustFunctionVisitor{})
}

type auditMustFunctionVisitor struct {
	visitor.GoVisitor
}

func (v *auditMustFunctionVisitor) VisitMethodInvocation(mi *tree.MethodInvocation, p any) tree.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*tree.MethodInvocation)

	if !strings.HasPrefix(mi.Name.Name, "Must") && !strings.HasPrefix(mi.Name.Name, "must") {
		return mi
	}

	mi = mi.WithMarkers(
		tree.MarkupInfo(mi.Markers, "Must* function panics on error; use with care"),
	)
	return mi
}
