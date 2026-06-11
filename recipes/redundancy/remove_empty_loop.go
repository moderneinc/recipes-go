/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package redundancy

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// RemoveEmptyLoop removes `for` loops with empty bodies. An empty loop body spins
// the CPU without doing useful work and is likely a bug or placeholder.
type RemoveEmptyLoop struct {
	recipe.Base
}

func (r *RemoveEmptyLoop) Name() string {
	return "org.openrewrite.golang.codequality.RemoveEmptyLoop"
}
func (r *RemoveEmptyLoop) DisplayName() string { return "Remove empty for loop" }
func (r *RemoveEmptyLoop) Description() string {
	return "Remove `for` loops with empty bodies that spin without doing useful work."
}
func (r *RemoveEmptyLoop) Tags() []string { return []string{"cleanup", "redundancy"} }

func (r *RemoveEmptyLoop) Editor() recipe.TreeVisitor {
	return visitor.Init(&removeEmptyLoopVisitor{})
}

type removeEmptyLoopVisitor struct {
	visitor.GoVisitor
}

func (v *removeEmptyLoopVisitor) VisitForLoop(forLoop *java.ForLoop, p any) java.J {
	forLoop = v.GoVisitor.VisitForLoop(forLoop, p).(*java.ForLoop)

	if !isEmptyBlock(forLoop.Body) {
		return forLoop
	}

	// Remove the empty for loop by replacing with Empty.
	return &java.Empty{}
}

func (v *removeEmptyLoopVisitor) VisitForEachLoop(forEach *java.ForEachLoop, p any) java.J {
	forEach = v.GoVisitor.VisitForEachLoop(forEach, p).(*java.ForEachLoop)

	if !isEmptyBlock(forEach.Body) {
		return forEach
	}

	// Remove the empty for-range loop by replacing with Empty.
	return &java.Empty{}
}

// isEmptyBlock returns true if the block is nil or contains no real statements
// (only Empty sentinels).
func isEmptyBlock(block *java.Block) bool {
	if block == nil {
		return true
	}
	for _, stmt := range block.Statements {
		if _, isEmpty := stmt.Element.(*java.Empty); !isEmpty {
			return false
		}
	}
	return true
}
