/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindDeeplyNestedBlock finds blocks nested 4 or more levels deep
// (e.g. if inside if inside if inside for). Deep nesting indicates
// the need for early returns or function extraction.
// golangci-lint: nestif
type FindDeeplyNestedBlock struct {
	recipe.Base
}

func (r *FindDeeplyNestedBlock) Name() string {
	return "org.openrewrite.golang.codequality.FindDeeplyNestedBlock"
}
func (r *FindDeeplyNestedBlock) DisplayName() string { return "Find deeply nested blocks" }
func (r *FindDeeplyNestedBlock) Description() string {
	return "Find blocks nested 4 or more levels deep. Deep nesting makes code harder to follow; consider early returns or extracting helper functions."
}
func (r *FindDeeplyNestedBlock) Tags() []string { return []string{"style", "lint"} }

func (r *FindDeeplyNestedBlock) Editor() recipe.TreeVisitor {
	return visitor.Init(&findDeeplyNestedBlockVisitor{})
}

type findDeeplyNestedBlockVisitor struct {
	visitor.GoVisitor
	blockDepth int
}

func (v *findDeeplyNestedBlockVisitor) VisitBlock(block *tree.Block, p any) tree.J {
	v.blockDepth++
	block = v.GoVisitor.VisitBlock(block, p).(*tree.Block)
	depth := v.blockDepth
	v.blockDepth--

	if depth < 4 {
		return block
	}

	block = block.WithMarkers(
		tree.FoundSearchResult(block.Markers, "deeply nested block"),
	)
	return block
}
