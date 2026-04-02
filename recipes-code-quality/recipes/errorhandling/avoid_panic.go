/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// AvoidPanic finds calls to the built-in `panic()` function. Panics crash the
// entire program and should generally be avoided in library code and goroutines.
type AvoidPanic struct {
	recipe.Base
}

func (r *AvoidPanic) Name() string {
	return "org.openrewrite.golang.codequality.AvoidPanic"
}
func (r *AvoidPanic) DisplayName() string { return "Avoid panic" }
func (r *AvoidPanic) Description() string {
	return "Find calls to the built-in `panic()` function, which crashes the program."
}
func (r *AvoidPanic) Tags() []string { return []string{"errorhandling", "lint"} }

func (r *AvoidPanic) Editor() recipe.TreeVisitor {
	return visitor.Init(&avoidPanicVisitor{})
}

type avoidPanicVisitor struct {
	visitor.GoVisitor
}

func (v *avoidPanicVisitor) VisitMethodInvocation(mi *tree.MethodInvocation, p any) tree.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*tree.MethodInvocation)

	// Match: panic(...) — built-in, so no Select and Name == "panic".
	if mi.Select != nil || mi.Name.Name != "panic" {
		return mi
	}

	mi = mi.WithMarkers(
		tree.MarkupWarn(mi.Markers, "panic call found; consider returning an error instead"),
	)
	return mi
}
