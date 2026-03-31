/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindJsonNumber finds usage of `json.Number`. The json.Number type should
// be used carefully as it can lead to unexpected behavior when converting
// between numeric types.
type FindJsonNumber struct {
	recipe.Base
}

func (r *FindJsonNumber) Name() string {
	return "org.openrewrite.golang.codequality.FindJsonNumber"
}
func (r *FindJsonNumber) DisplayName() string { return "Find json.Number usage" }
func (r *FindJsonNumber) Description() string {
	return "Find usage of `json.Number`. json.Number should be used carefully as it can lead to unexpected behavior when converting between numeric types."
}
func (r *FindJsonNumber) Tags() []string { return []string{"style"} }

func (r *FindJsonNumber) Editor() recipe.TreeVisitor {
	return visitor.Init(&findJsonNumberVisitor{})
}

type findJsonNumberVisitor struct {
	visitor.GoVisitor
}

func (v *findJsonNumberVisitor) VisitFieldAccess(fa *tree.FieldAccess, p any) tree.J {
	fa = v.GoVisitor.VisitFieldAccess(fa, p).(*tree.FieldAccess)

	ident, ok := fa.Target.(*tree.Identifier)
	if !ok || ident.Name != "json" {
		return fa
	}

	if fa.Name.Element.Name != "Number" {
		return fa
	}

	fa = fa.WithMarkers(tree.FoundSearchResult(fa.Markers, "json.Number should be used carefully"))
	return fa
}
