/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// AvoidFallthrough removes fallthrough statements in switch cases.
// Fallthrough is rarely used in Go and can be confusing to readers.
// Removing fallthrough changes behavior by stopping the fall-through,
// which is the intended remediation.
type AvoidFallthrough struct {
	recipe.Base
}

func (r *AvoidFallthrough) Name() string {
	return "org.openrewrite.golang.codequality.AvoidFallthrough"
}
func (r *AvoidFallthrough) DisplayName() string { return "Avoid fallthrough" }
func (r *AvoidFallthrough) Description() string {
	return "Remove fallthrough statements in switch cases. Fallthrough is rarely used in Go and can be confusing."
}
func (r *AvoidFallthrough) Tags() []string { return []string{"style"} }

func (r *AvoidFallthrough) Editor() recipe.TreeVisitor {
	return visitor.Init(&avoidFallthroughVisitor{})
}

type avoidFallthroughVisitor struct {
	visitor.GoVisitor
}

func (v *avoidFallthroughVisitor) VisitCase(c *java.Case, p any) java.J {
	c = v.GoVisitor.VisitCase(c, p).(*java.Case)

	// Remove any fallthrough statements from the case body.
	changed := false
	var body []java.RightPadded[java.Statement]
	for _, rp := range c.Body {
		if _, ok := rp.Element.(*golang.Fallthrough); ok {
			changed = true
			continue
		}
		body = append(body, rp)
	}

	if !changed {
		return c
	}

	c.Body = body
	return c
}
