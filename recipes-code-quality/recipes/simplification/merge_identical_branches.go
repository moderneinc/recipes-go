/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/printer"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// MergeIdenticalBranches merges consecutive if/else-if branches that have
// identical bodies by combining their conditions with `||`.
//
//	if a { x() } else if b { x() } else { y() }
//	// becomes
//	if a || b { x() } else { y() }
//
// Only adjacent branches with identical bodies are merged. If the two
// conditions are the last remaining branches (no further else), the else-if
// is dropped entirely.
type MergeIdenticalBranches struct {
	recipe.Base
}

func (r *MergeIdenticalBranches) Name() string {
	return "org.openrewrite.golang.codequality.MergeIdenticalBranches"
}
func (r *MergeIdenticalBranches) DisplayName() string { return "Merge identical branches" }
func (r *MergeIdenticalBranches) Description() string {
	return "Merge consecutive if/else-if branches that have identical bodies by combining " +
		"their conditions with `||`."
}
func (r *MergeIdenticalBranches) Tags() []string {
	return []string{"cleanup", "simplification", "RSPEC-S1871"}
}

func (r *MergeIdenticalBranches) Editor() recipe.TreeVisitor {
	return visitor.Init(&mergeIdenticalBranchesVisitor{})
}

type mergeIdenticalBranchesVisitor struct {
	visitor.GoVisitor
}

func (v *mergeIdenticalBranchesVisitor) VisitIf(ifStmt *java.If, p any) java.J {
	ifStmt = v.GoVisitor.VisitIf(ifStmt, p).(*java.If)

	result := mergeAdjacentBranches(ifStmt)
	if result == nil {
		return ifStmt
	}
	return result
}

// mergeAdjacentBranches walks the if/else-if chain and merges adjacent branches
// with identical bodies. Returns nil if nothing changed.
func mergeAdjacentBranches(ifStmt *java.If) *java.If {
	changed := false
	current := ifStmt

	for current.ElsePart != nil {
		nextIf, ok := current.ElsePart.Element.(*java.If)
		if !ok {
			// Plain else block — can't merge further.
			break
		}

		if !bodiesEqual(current.Then, nextIf.Then) {
			current = nextIf
			continue
		}

		// Merge: combine conditions with ||
		combined := &java.Binary{
			Left:     current.Condition,
			Operator: java.LeftPadded[java.BinaryOperator]{Before: java.SingleSpace, Element: java.LogicalOr},
			Right:    setExprPrefix(nextIf.Condition, java.SingleSpace),
		}

		current.Condition = combined
		// Skip the duplicate branch — adopt its else part.
		current.ElsePart = nextIf.ElsePart
		changed = true
		// Don't advance — check if the new next branch also matches.
	}

	if !changed {
		return nil
	}
	return ifStmt
}

// bodiesEqual checks whether two blocks have the same printed representation
// (ignoring leading whitespace differences).
func bodiesEqual(a, b *java.Block) bool {
	if a == nil || b == nil {
		return a == b
	}
	return printBlock(a) == printBlock(b)
}

func printBlock(block *java.Block) string {
	return printer.Print(block.WithPrefix(java.Space{}))
}
