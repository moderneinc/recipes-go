/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package redundancy

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindEmptyLoop finds `for` loops with empty bodies. An empty loop body spins
// the CPU without doing useful work and is likely a bug or placeholder.
type FindEmptyLoop struct {
	recipe.Base
}

func (r *FindEmptyLoop) Name() string {
	return "org.openrewrite.golang.codequality.FindEmptyLoop"
}
func (r *FindEmptyLoop) DisplayName() string { return "Find empty for loop" }
func (r *FindEmptyLoop) Description() string {
	return "Find `for` loops with empty bodies that spin without doing useful work."
}
func (r *FindEmptyLoop) Tags() []string { return []string{"cleanup", "redundancy"} }

func (r *FindEmptyLoop) Editor() recipe.TreeVisitor {
	return visitor.Init(&findEmptyLoopVisitor{})
}

type findEmptyLoopVisitor struct {
	visitor.GoVisitor
}

func (v *findEmptyLoopVisitor) VisitForLoop(forLoop *tree.ForLoop, p any) tree.J {
	forLoop = v.GoVisitor.VisitForLoop(forLoop, p).(*tree.ForLoop)

	if !isEmptyBlock(forLoop.Body) {
		return forLoop
	}

	forLoop = forLoop.WithMarkers(
		tree.FoundSearchResult(forLoop.Markers, "empty for loop body"),
	)
	return forLoop
}

func (v *findEmptyLoopVisitor) VisitForEachLoop(forEach *tree.ForEachLoop, p any) tree.J {
	forEach = v.GoVisitor.VisitForEachLoop(forEach, p).(*tree.ForEachLoop)

	if !isEmptyBlock(forEach.Body) {
		return forEach
	}

	forEach = forEach.WithMarkers(
		tree.FoundSearchResult(forEach.Markers, "empty for-range loop body"),
	)
	return forEach
}

// isEmptyBlock returns true if the block is nil or contains no real statements
// (only Empty sentinels).
func isEmptyBlock(block *tree.Block) bool {
	if block == nil {
		return true
	}
	for _, stmt := range block.Statements {
		if _, isEmpty := stmt.Element.(*tree.Empty); !isEmpty {
			return false
		}
	}
	return true
}
