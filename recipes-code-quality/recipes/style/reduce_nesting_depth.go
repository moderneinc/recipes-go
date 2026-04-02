/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// ReduceNestingDepth finds blocks nested 4 or more levels deep
// (e.g. if inside if inside if inside for). Deep nesting indicates
// the need for early returns or function extraction.
// golangci-lint: nestif
type ReduceNestingDepth struct {
	recipe.Base
}

func (r *ReduceNestingDepth) Name() string {
	return "org.openrewrite.golang.codequality.ReduceNestingDepth"
}
func (r *ReduceNestingDepth) DisplayName() string { return "Reduce nesting depth" }
func (r *ReduceNestingDepth) Description() string {
	return "Find blocks nested 4 or more levels deep. Deep nesting makes code harder to follow; consider early returns or extracting helper functions."
}
func (r *ReduceNestingDepth) Tags() []string { return []string{"style", "lint"} }

func (r *ReduceNestingDepth) Editor() recipe.TreeVisitor {
	return visitor.Init(&reduceNestingDepthVisitor{})
}

type reduceNestingDepthVisitor struct {
	visitor.GoVisitor
	blockDepth int
}

func (v *reduceNestingDepthVisitor) VisitBlock(block *tree.Block, p any) tree.J {
	v.blockDepth++
	block = v.GoVisitor.VisitBlock(block, p).(*tree.Block)
	depth := v.blockDepth
	v.blockDepth--

	if depth < 4 {
		return block
	}

	block = block.WithMarkers(
		tree.MarkupWarn(block.Markers, "deeply nested block"),
	)
	return block
}
