/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindManyReturns finds functions with more than 3 return values.
// Too many return values make the call site unwieldy; consider returning
// a struct instead.
type FindManyReturns struct {
	recipe.Base
}

func (r *FindManyReturns) Name() string {
	return "org.openrewrite.golang.codequality.FindManyReturns"
}
func (r *FindManyReturns) DisplayName() string { return "Find functions with too many return values" }
func (r *FindManyReturns) Description() string {
	return "Find functions with more than 3 return values. Consider returning a struct instead."
}
func (r *FindManyReturns) Tags() []string { return []string{"style", "lint"} }

func (r *FindManyReturns) Editor() recipe.TreeVisitor {
	return visitor.Init(&findManyReturnsVisitor{})
}

type findManyReturnsVisitor struct {
	visitor.GoVisitor
}

func (v *findManyReturnsVisitor) VisitMethodDeclaration(md *tree.MethodDeclaration, p any) tree.J {
	md = v.GoVisitor.VisitMethodDeclaration(md, p).(*tree.MethodDeclaration)

	if md.Name == nil || md.ReturnType == nil {
		return md
	}

	tl, ok := md.ReturnType.(*tree.TypeList)
	if !ok {
		// Single return value -- not a problem.
		return md
	}

	count := 0
	for _, elem := range tl.Types.Elements {
		if _, isEmpty := elem.Element.(*tree.Empty); !isEmpty {
			count++
		}
	}

	if count <= 3 {
		return md
	}

	md = md.WithName(md.Name.WithMarkers(
		tree.FoundSearchResult(md.Name.Markers, "function has too many return values"),
	))
	return md
}
