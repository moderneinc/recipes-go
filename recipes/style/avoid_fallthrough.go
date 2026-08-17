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

// AvoidFallthrough marks fallthrough statements in switch cases. It reports
// rather than rewrites: the remediation is to merge or duplicate the cases,
// which changes the code around the statement.
type AvoidFallthrough struct {
	recipe.Base
}

func (r *AvoidFallthrough) Name() string {
	return "org.openrewrite.golang.codequality.AvoidFallthrough"
}
func (r *AvoidFallthrough) DisplayName() string { return "Avoid fallthrough" }
func (r *AvoidFallthrough) Description() string {
	return "Find fallthrough statements in switch cases. Fallthrough is rarely used in Go and can be confusing."
}
func (r *AvoidFallthrough) Tags() []string { return []string{"style"} }

func (r *AvoidFallthrough) Editor() recipe.TreeVisitor {
	return visitor.Init(&avoidFallthroughVisitor{})
}

type avoidFallthroughVisitor struct {
	visitor.GoVisitor
}

func (v *avoidFallthroughVisitor) VisitFallthrough(f *golang.Fallthrough, p any) java.J {
	f = v.GoVisitor.VisitFallthrough(f, p).(*golang.Fallthrough)
	return f.WithMarkers(java.MarkupInfo(f.Markers,
		"fallthrough continues into the next case; merge or duplicate the cases instead"))
}
