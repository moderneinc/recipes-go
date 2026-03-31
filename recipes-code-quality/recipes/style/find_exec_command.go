/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindExecCommand finds calls to `exec.Command()`. If command arguments come
// from user input, this is a potential command injection vulnerability. Validate
// and sanitize all inputs before passing them to exec.Command.
type FindExecCommand struct {
	recipe.Base
}

func (r *FindExecCommand) Name() string {
	return "org.openrewrite.golang.codequality.FindExecCommand"
}
func (r *FindExecCommand) DisplayName() string { return "Find exec.Command calls" }
func (r *FindExecCommand) Description() string {
	return "Find calls to `exec.Command()`. If arguments come from user input, this is a potential command injection vulnerability."
}
func (r *FindExecCommand) Tags() []string { return []string{"security"} }

func (r *FindExecCommand) Editor() recipe.TreeVisitor {
	return visitor.Init(&findExecCommandVisitor{})
}

type findExecCommandVisitor struct {
	visitor.GoVisitor
}

func (v *findExecCommandVisitor) VisitMethodInvocation(mi *tree.MethodInvocation, p any) tree.J {
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

	mi = mi.WithMarkers(tree.FoundSearchResult(mi.Markers, "exec.Command call; ensure arguments are not from untrusted input"))
	return mi
}
