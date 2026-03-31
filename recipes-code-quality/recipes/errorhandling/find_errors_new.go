/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindErrorsNew finds all calls to `errors.New(...)`. Inline error sentinels
// created with errors.New inside function bodies cannot be compared with
// errors.Is. Consider assigning them to package-level variables instead.
type FindErrorsNew struct {
	recipe.Base
}

func (r *FindErrorsNew) Name() string {
	return "org.openrewrite.golang.codequality.FindErrorsNew"
}
func (r *FindErrorsNew) DisplayName() string { return "Find errors.New calls" }
func (r *FindErrorsNew) Description() string {
	return "Find all `errors.New(...)` calls. Inline error sentinels cannot be compared with `errors.Is`; consider assigning them to package-level variables."
}
func (r *FindErrorsNew) Tags() []string { return []string{"error-handling", "lint"} }

func (r *FindErrorsNew) Editor() recipe.TreeVisitor {
	return visitor.Init(&findErrorsNewVisitor{})
}

type findErrorsNewVisitor struct {
	visitor.GoVisitor
}

func (v *findErrorsNewVisitor) VisitMethodInvocation(mi *tree.MethodInvocation, p any) tree.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*tree.MethodInvocation)

	if mi.Select == nil {
		return mi
	}

	ident, ok := mi.Select.Element.(*tree.Identifier)
	if !ok || ident.Name != "errors" {
		return mi
	}

	if mi.Name.Name != "New" {
		return mi
	}

	mi = mi.WithMarkers(
		tree.FoundSearchResult(mi.Markers, "errors.New call found; consider using a package-level sentinel variable"),
	)
	return mi
}
