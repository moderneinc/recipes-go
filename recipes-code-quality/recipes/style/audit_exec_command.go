/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// AuditExecCommand finds calls to `exec.Command()`. If command arguments come
// from user input, this is a potential command injection vulnerability. Validate
// and sanitize all inputs before passing them to exec.Command.
type AuditExecCommand struct {
	recipe.Base
}

func (r *AuditExecCommand) Name() string {
	return "org.openrewrite.golang.codequality.AuditExecCommand"
}
func (r *AuditExecCommand) DisplayName() string { return "Audit exec.Command calls" }
func (r *AuditExecCommand) Description() string {
	return "Find calls to `exec.Command()`. If arguments come from user input, this is a potential command injection vulnerability."
}
func (r *AuditExecCommand) Tags() []string { return []string{"security"} }

func (r *AuditExecCommand) Editor() recipe.TreeVisitor {
	return visitor.Init(&auditExecCommandVisitor{})
}

type auditExecCommandVisitor struct {
	visitor.GoVisitor
}

func (v *auditExecCommandVisitor) VisitMethodInvocation(mi *tree.MethodInvocation, p any) tree.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*tree.MethodInvocation)

	if mi.Select == nil {
		return mi
	}

	ident, ok := mi.Select.Element.(*tree.Identifier)
	if !ok || ident.Name != "exec" {
		return mi
	}

	if mi.Name.Name != "Command" {
		return mi
	}

	mi = mi.WithMarkers(tree.MarkupWarn(mi.Markers, "exec.Command call; ensure arguments are not from untrusted input"))
	return mi
}
