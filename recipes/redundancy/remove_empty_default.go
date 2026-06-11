/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package redundancy

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// RemoveEmptyDefault removes `default:` cases with empty bodies from switch
// statements. An empty default clause is typically dead code or a
// placeholder that was never filled in.
type RemoveEmptyDefault struct {
	recipe.Base
}

func (r *RemoveEmptyDefault) Name() string {
	return "org.openrewrite.golang.codequality.RemoveEmptyDefault"
}
func (r *RemoveEmptyDefault) DisplayName() string { return "Remove empty default case" }
func (r *RemoveEmptyDefault) Description() string {
	return "Remove `default:` cases with empty bodies from switch statements."
}
func (r *RemoveEmptyDefault) Tags() []string { return []string{"cleanup", "redundancy"} }

func (r *RemoveEmptyDefault) Editor() recipe.TreeVisitor {
	return visitor.Init(&removeEmptyDefaultVisitor{})
}

type removeEmptyDefaultVisitor struct {
	visitor.GoVisitor
}

func (v *removeEmptyDefaultVisitor) VisitCase(c *java.Case, p any) java.J {
	c = v.GoVisitor.VisitCase(c, p).(*java.Case)

	// Must be a default case (no expressions).
	if len(c.Expressions.Elements) != 0 {
		return c
	}

	// Body must have no real statements (only Empty sentinels count as empty).
	for _, stmt := range c.Body {
		if _, ok := stmt.Element.(*java.Empty); !ok {
			return c
		}
	}

	// Remove the empty default case by replacing with Empty.
	return &java.Empty{}
}
