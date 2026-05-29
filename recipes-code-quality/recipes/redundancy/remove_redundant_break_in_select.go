/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package redundancy

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// RemoveRedundantBreakInSelect removes trailing `break` statements at the end
// of select communication clauses. Go select cases do not fall through, so a
// trailing break is redundant.
type RemoveRedundantBreakInSelect struct {
	recipe.Base
}

func (r *RemoveRedundantBreakInSelect) Name() string {
	return "org.openrewrite.golang.codequality.RemoveRedundantBreakInSelect"
}
func (r *RemoveRedundantBreakInSelect) DisplayName() string {
	return "Remove redundant break in select"
}
func (r *RemoveRedundantBreakInSelect) Description() string {
	return "Remove trailing `break` in select communication clauses. Go select cases do not fall through."
}
func (r *RemoveRedundantBreakInSelect) Tags() []string {
	return []string{"cleanup", "simplification"}
}

func (r *RemoveRedundantBreakInSelect) Editor() recipe.TreeVisitor {
	return visitor.Init(&removeRedundantBreakInSelectVisitor{})
}

type removeRedundantBreakInSelectVisitor struct {
	visitor.GoVisitor
}

func (v *removeRedundantBreakInSelectVisitor) VisitCommClause(cc *golang.CommClause, p any) java.J {
	cc = v.GoVisitor.VisitCommClause(cc, p).(*golang.CommClause)

	if len(cc.Body) == 0 {
		return cc
	}

	// Check if the last statement is a break with no label.
	last := cc.Body[len(cc.Body)-1]
	brk, ok := last.Element.(*java.Break)
	if !ok || brk.Label != nil {
		return cc
	}

	// Remove the trailing break.
	cc.Body = cc.Body[:len(cc.Body)-1]
	return cc
}
