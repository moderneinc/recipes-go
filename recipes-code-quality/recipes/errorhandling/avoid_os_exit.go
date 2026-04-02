/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// AvoidOsExit removes `os.Exit(0)` calls (the program exits naturally at end
// of main) and marks non-zero `os.Exit(n)` calls with a warning, since those
// bypass deferred functions and cleanup.
type AvoidOsExit struct {
	recipe.Base
}

func (r *AvoidOsExit) Name() string {
	return "org.openrewrite.golang.codequality.AvoidOsExit"
}
func (r *AvoidOsExit) DisplayName() string { return "Avoid os.Exit" }
func (r *AvoidOsExit) Description() string {
	return "Remove `os.Exit(0)` calls and flag non-zero `os.Exit(n)` which bypass deferred functions and cleanup."
}
func (r *AvoidOsExit) Tags() []string { return []string{"error-handling", "lint"} }

func (r *AvoidOsExit) Editor() recipe.TreeVisitor {
	return visitor.Init(&avoidOsExitVisitor{})
}

type avoidOsExitVisitor struct {
	visitor.GoVisitor
}

func (v *avoidOsExitVisitor) VisitMethodInvocation(mi *tree.MethodInvocation, p any) tree.J {
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

	// os.Exit(0) can be safely removed — the program exits naturally.
	args := mi.Arguments.Elements
	if len(args) == 1 {
		if lit, ok := args[0].Element.(*tree.Literal); ok && lit.Source == "0" {
			return &tree.Empty{}
		}
	}

	// Non-zero exit codes: keep but warn.
	mi = mi.WithMarkers(
		tree.MarkupWarn(mi.Markers, "os.Exit bypasses deferred functions and cleanup"),
	)
	return mi
}
