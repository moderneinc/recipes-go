/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindEmptyFunction finds functions with empty bodies (no statements).
// These are usually stubs or forgotten implementations.
type FindEmptyFunction struct {
	recipe.Base
}

func (r *FindEmptyFunction) Name() string {
	return "org.openrewrite.golang.codequality.FindEmptyFunction"
}
func (r *FindEmptyFunction) DisplayName() string { return "Find empty functions" }
func (r *FindEmptyFunction) Description() string {
	return "Find functions with empty bodies. Empty functions are usually stubs or forgotten implementations."
}
func (r *FindEmptyFunction) Tags() []string { return []string{"style", "lint"} }

func (r *FindEmptyFunction) Editor() recipe.TreeVisitor {
	return visitor.Init(&findEmptyFunctionVisitor{})
}

type findEmptyFunctionVisitor struct {
	visitor.GoVisitor
}

func (v *findEmptyFunctionVisitor) VisitMethodDeclaration(md *tree.MethodDeclaration, p any) tree.J {
	md = v.GoVisitor.VisitMethodDeclaration(md, p).(*tree.MethodDeclaration)

	if md.Name == nil || md.Body == nil {
		return md
	}

	// Check if the body has any real statements (not just Empty sentinels).
	for _, stmt := range md.Body.Statements {
		if _, isEmpty := stmt.Element.(*tree.Empty); !isEmpty {
			return md
		}
	}

	md = md.WithName(md.Name.WithMarkers(
		tree.FoundSearchResult(md.Name.Markers, "empty function body"),
	))
	return md
}
