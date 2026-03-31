/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindErrorFmtSprint finds `fmt.Sprint(err)` calls. Converting an error to a
// string via fmt.Sprint is unclear — use `err.Error()` for direct access or
// `fmt.Errorf` for wrapping with context.
type FindErrorFmtSprint struct {
	recipe.Base
}

func (r *FindErrorFmtSprint) Name() string {
	return "org.openrewrite.golang.codequality.FindErrorFmtSprint"
}
func (r *FindErrorFmtSprint) DisplayName() string { return "Find fmt.Sprint(err) calls" }
func (r *FindErrorFmtSprint) Description() string {
	return "Find `fmt.Sprint(err)` calls. Use `err.Error()` for clarity or `fmt.Errorf` for wrapping."
}
func (r *FindErrorFmtSprint) Tags() []string { return []string{"error-handling", "lint"} }

func (r *FindErrorFmtSprint) Editor() recipe.TreeVisitor {
	return visitor.Init(&findErrorFmtSprintVisitor{})
}

type findErrorFmtSprintVisitor struct {
	visitor.GoVisitor
}

func (v *findErrorFmtSprintVisitor) VisitMethodInvocation(mi *tree.MethodInvocation, p any) tree.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*tree.MethodInvocation)

	if mi.Select == nil {
		return mi
	}

	ident, ok := mi.Select.Element.(*tree.Identifier)
	if !ok || ident.Name != "fmt" {
		return mi
	}

	if mi.Name.Name != "Sprint" {
		return mi
	}

	// Check for exactly 1 real argument that is an identifier named "err".
	args := realArgs(mi.Arguments.Elements)
	if len(args) != 1 {
		return mi
	}

	argIdent, ok := args[0].Element.(*tree.Identifier)
	if !ok || argIdent.Name != "err" {
		return mi
	}

	mi = mi.WithMarkers(
		tree.FoundSearchResult(mi.Markers, "use err.Error() instead of fmt.Sprint(err)"),
	)
	return mi
}

// realArgs filters out Empty sentinel elements from an argument list.
func realArgs(elems []tree.RightPadded[tree.Expression]) []tree.RightPadded[tree.Expression] {
	var result []tree.RightPadded[tree.Expression]
	for _, e := range elems {
		if _, isEmpty := e.Element.(*tree.Empty); !isEmpty {
			result = append(result, e)
		}
	}
	return result
}
