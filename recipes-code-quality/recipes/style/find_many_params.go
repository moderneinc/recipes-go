/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindManyParams finds functions with more than 5 parameters.
// Too many parameters suggest the function should accept a struct instead.
type FindManyParams struct {
	recipe.Base
}

func (r *FindManyParams) Name() string {
	return "org.openrewrite.golang.codequality.FindManyParams"
}
func (r *FindManyParams) DisplayName() string { return "Find functions with too many parameters" }
func (r *FindManyParams) Description() string {
	return "Find functions with more than 5 parameters. Consider grouping parameters into a struct."
}
func (r *FindManyParams) Tags() []string { return []string{"style", "lint"} }

func (r *FindManyParams) Editor() recipe.TreeVisitor {
	return visitor.Init(&findManyParamsVisitor{})
}

type findManyParamsVisitor struct {
	visitor.GoVisitor
}

func (v *findManyParamsVisitor) VisitMethodDeclaration(md *tree.MethodDeclaration, p any) tree.J {
	md = v.GoVisitor.VisitMethodDeclaration(md, p).(*tree.MethodDeclaration)

	if md.Name == nil {
		return md
	}

	count := 0
	for _, param := range md.Parameters.Elements {
		if _, isEmpty := param.Element.(*tree.Empty); !isEmpty {
			count++
		}
	}

	if count <= 5 {
		return md
	}

	md = md.WithName(md.Name.WithMarkers(
		tree.FoundSearchResult(md.Name.Markers, "function has too many parameters"),
	))
	return md
}
