/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindSkipWithoutReason finds `t.Skip()` calls that have no message argument.
// Tests should document why they are skipped so that the reason is visible in
// test output and code review.
type FindSkipWithoutReason struct {
	recipe.Base
}

func (r *FindSkipWithoutReason) Name() string {
	return "org.openrewrite.golang.codequality.FindSkipWithoutReason"
}
func (r *FindSkipWithoutReason) DisplayName() string { return "Find t.Skip() without reason" }
func (r *FindSkipWithoutReason) Description() string {
	return "Find `t.Skip()` calls without a message argument. Tests should document why they are skipped."
}
func (r *FindSkipWithoutReason) Tags() []string { return []string{"testing"} }

func (r *FindSkipWithoutReason) Editor() recipe.TreeVisitor {
	return visitor.Init(&findSkipWithoutReasonVisitor{})
}

type findSkipWithoutReasonVisitor struct {
	visitor.GoVisitor
}

func (v *findSkipWithoutReasonVisitor) VisitMethodInvocation(mi *tree.MethodInvocation, p any) tree.J {
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

	mi = mi.WithMarkers(
		tree.FoundSearchResult(mi.Markers, "t.Skip() without reason; add a message explaining why"),
	)
	return mi
}
