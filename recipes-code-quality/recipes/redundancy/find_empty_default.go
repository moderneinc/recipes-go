/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package redundancy

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindEmptyDefault finds `default:` cases with empty bodies in switch
// statements. An empty default clause is typically dead code or a
// placeholder that was never filled in.
type FindEmptyDefault struct {
	recipe.Base
}

func (r *FindEmptyDefault) Name() string {
	return "org.openrewrite.golang.codequality.FindEmptyDefault"
}
func (r *FindEmptyDefault) DisplayName() string { return "Find empty default case" }
func (r *FindEmptyDefault) Description() string {
	return "Find `default:` cases with empty bodies in switch statements."
}
func (r *FindEmptyDefault) Tags() []string { return []string{"cleanup", "redundancy"} }

func (r *FindEmptyDefault) Editor() recipe.TreeVisitor {
	return visitor.Init(&findEmptyDefaultVisitor{})
}

type findEmptyDefaultVisitor struct {
	visitor.GoVisitor
}

func (v *findEmptyDefaultVisitor) VisitCase(c *tree.Case, p any) tree.J {
	c = v.GoVisitor.VisitCase(c, p).(*tree.Case)

	// Must be a default case (no expressions).
	if len(c.Expressions.Elements) != 0 {
		return c
	}

	// Body must have no real statements (only Empty sentinels count as empty).
	for _, stmt := range c.Body {
		if _, ok := stmt.Element.(*tree.Empty); !ok {
			return c
		}
	}

	c = c.WithMarkers(
		tree.FoundSearchResult(c.Markers, "empty default case"),
	)
	return c
}
