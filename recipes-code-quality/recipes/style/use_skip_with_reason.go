/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// UseSkipWithReason adds a placeholder reason to bare `t.Skip()` calls.
// Tests should document why they are skipped so that the reason is visible in
// test output and code review. Transforms `t.Skip()` into
// `t.Skip("TODO: add reason")`.
type UseSkipWithReason struct {
	recipe.Base
}

func (r *UseSkipWithReason) Name() string {
	return "org.openrewrite.golang.codequality.UseSkipWithReason"
}
func (r *UseSkipWithReason) DisplayName() string { return "Use skip with reason" }
func (r *UseSkipWithReason) Description() string {
	return "Add a placeholder reason to bare `t.Skip()` calls. Tests should document why they are skipped."
}
func (r *UseSkipWithReason) Tags() []string { return []string{"testing"} }

func (r *UseSkipWithReason) Editor() recipe.TreeVisitor {
	return visitor.Init(&useSkipWithReasonVisitor{})
}

type useSkipWithReasonVisitor struct {
	visitor.GoVisitor
}

func (v *useSkipWithReasonVisitor) VisitMethodInvocation(mi *tree.MethodInvocation, p any) tree.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*tree.MethodInvocation)

	if mi.Select == nil {
		return mi
	}

	ident, ok := mi.Select.Element.(*tree.Identifier)
	if !ok || ident.Name != "t" {
		return mi
	}

	if mi.Name.Name != "Skip" {
		return mi
	}

	// Check if there are any real arguments (skip Empty sentinels).
	for _, arg := range mi.Arguments.Elements {
		if _, isEmpty := arg.Element.(*tree.Empty); !isEmpty {
			return mi
		}
	}

	// Replace the empty argument list with a placeholder reason string.
	reason := &tree.Literal{
		Kind:   tree.StringLiteral,
		Value:  "TODO: add reason",
		Source: `"TODO: add reason"`,
	}
	args := mi.Arguments
	args.Elements = []tree.RightPadded[tree.Expression]{
		{Element: reason},
	}
	mi.Arguments = args
	return mi
}
