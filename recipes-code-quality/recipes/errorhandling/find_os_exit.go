/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindOsExit finds calls to `os.Exit()`. These bypass deferred functions and
// cleanup. Should usually only appear in main().
type FindOsExit struct {
	recipe.Base
}

func (r *FindOsExit) Name() string {
	return "org.openrewrite.golang.codequality.FindOsExit"
}
func (r *FindOsExit) DisplayName() string { return "Find os.Exit calls" }
func (r *FindOsExit) Description() string {
	return "Find `os.Exit()` calls which bypass deferred functions and cleanup."
}
func (r *FindOsExit) Tags() []string { return []string{"error-handling", "lint"} }

func (r *FindOsExit) Editor() recipe.TreeVisitor {
	return visitor.Init(&findOsExitVisitor{})
}

type findOsExitVisitor struct {
	visitor.GoVisitor
}

func (v *findOsExitVisitor) VisitMethodInvocation(mi *tree.MethodInvocation, p any) tree.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*tree.MethodInvocation)

	if mi.Select == nil {
		return mi
	}

	ident, ok := mi.Select.Element.(*tree.Identifier)
	if !ok || ident.Name != "os" {
		return mi
	}

	if mi.Name.Name != "Exit" {
		return mi
	}

	mi = mi.WithMarkers(
		tree.FoundSearchResult(mi.Markers, "os.Exit bypasses deferred functions and cleanup"),
	)
	return mi
}
