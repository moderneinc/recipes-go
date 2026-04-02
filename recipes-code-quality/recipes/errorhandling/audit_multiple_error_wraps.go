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

// AuditMultipleErrorWraps finds `fmt.Errorf` calls whose format string contains
// more than one `%w` verb. Multiple `%w` verbs are invalid in Go < 1.20 and,
// while technically supported in Go 1.20+, are unusual and worth flagging for
// review.
type AuditMultipleErrorWraps struct {
	recipe.Base
}

func (r *AuditMultipleErrorWraps) Name() string {
	return "org.openrewrite.golang.codequality.AuditMultipleErrorWraps"
}
func (r *AuditMultipleErrorWraps) DisplayName() string {
	return "Audit multiple error wraps"
}
func (r *AuditMultipleErrorWraps) Description() string {
	return "Find `fmt.Errorf` calls whose format string contains more than one `%w` verb. Multiple error wrapping is invalid in Go < 1.20 and rare in later versions."
}
func (r *AuditMultipleErrorWraps) Tags() []string { return []string{"errorhandling", "lint"} }

func (r *AuditMultipleErrorWraps) Editor() recipe.TreeVisitor {
	return visitor.Init(&auditMultipleErrorWrapsVisitor{})
}

type auditMultipleErrorWrapsVisitor struct {
	visitor.GoVisitor
}

func (v *auditMultipleErrorWrapsVisitor) VisitMethodInvocation(mi *tree.MethodInvocation, p any) tree.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*tree.MethodInvocation)

	// Match: fmt.Errorf(...)
	if mi.Select == nil {
		return mi
	}
	ident, ok := mi.Select.Element.(*tree.Identifier)
	if !ok || ident.Name != "fmt" {
		return mi
	}
	if mi.Name.Name != "Errorf" {
		return mi
	}

	// Need at least one argument: the format string.
	args := mi.Arguments.Elements
	if len(args) < 1 {
		return mi
	}

	// First argument must be a string literal.
	fmtLit, ok := args[0].Element.(*tree.Literal)
	if !ok || fmtLit.Kind != tree.StringLiteral {
		return mi
	}

	// Count occurrences of %w in the format string.
	if strings.Count(fmtLit.Source, "%w") <= 1 {
		return mi
	}

	mi = mi.WithMarkers(
		tree.MarkupInfo(mi.Markers, "fmt.Errorf format string contains multiple %w verbs"),
	)
	return mi
}
