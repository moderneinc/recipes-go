/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package redundancy

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// RemoveRedundantBreak removes trailing `break` statements at the end of
// switch case clauses. Go cases do not fall through by default, so a
// trailing break is redundant.
type RemoveRedundantBreak struct {
	recipe.Base
}

func (r *RemoveRedundantBreak) Name() string {
	return "org.openrewrite.golang.codequality.RemoveRedundantBreak"
}
func (r *RemoveRedundantBreak) DisplayName() string { return "Remove redundant break" }
func (r *RemoveRedundantBreak) Description() string {
	return "Remove trailing `break` in switch case clauses. Go switch cases do not fall through by default."
}
func (r *RemoveRedundantBreak) Tags() []string { return []string{"cleanup", "simplification"} }

func (r *RemoveRedundantBreak) Editor() recipe.TreeVisitor {
	return visitor.Init(&removeRedundantBreakVisitor{})
}

type removeRedundantBreakVisitor struct {
	visitor.GoVisitor
}

func (v *removeRedundantBreakVisitor) VisitCase(c *java.Case, p any) java.J {
	c = v.GoVisitor.VisitCase(c, p).(*java.Case)

	if len(c.Body) == 0 {
		return c
	}

	// Check if the last statement is a break with no label.
	last := c.Body[len(c.Body)-1]
	brk, ok := last.Element.(*java.Break)
	if !ok || brk.Label != nil {
		return c
	}

	// Remove the trailing break.
	c.Body = c.Body[:len(c.Body)-1]
	return c
}
