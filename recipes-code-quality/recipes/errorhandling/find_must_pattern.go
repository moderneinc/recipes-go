/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling

import (
	"strings"

	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindMustFunction finds calls to functions named `Must*` or `must*`. These
// typically panic on error and should be used carefully — generally only in
// package-level variable initialization, not in request-handling code.
type FindMustFunction struct {
	recipe.Base
}

func (r *FindMustFunction) Name() string {
	return "org.openrewrite.golang.codequality.FindMustFunction"
}
func (r *FindMustFunction) DisplayName() string { return "Find Must* function calls" }
func (r *FindMustFunction) Description() string {
	return "Find calls to functions named `Must*` or `must*`, which typically panic on error."
}
func (r *FindMustFunction) Tags() []string { return []string{"error-handling", "lint"} }

func (r *FindMustFunction) Editor() recipe.TreeVisitor {
	return visitor.Init(&findMustFunctionVisitor{})
}

type findMustFunctionVisitor struct {
	visitor.GoVisitor
}

func (v *findMustFunctionVisitor) VisitMethodInvocation(mi *tree.MethodInvocation, p any) tree.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*tree.MethodInvocation)

	if !strings.HasPrefix(mi.Name.Name, "Must") && !strings.HasPrefix(mi.Name.Name, "must") {
		return mi
	}

	mi = mi.WithMarkers(
		tree.FoundSearchResult(mi.Markers, "Must* function panics on error; use with care"),
	)
	return mi
}
