/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package redundancy

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/printer"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// AllBranchesIdentical replaces an if/else chain whose branches all contain
// identical code with just the body. When every branch does the same thing the
// condition is meaningless and adds unnecessary complexity.
//
//	if a { x() } else { x() }  ->  { x() }
//	if a { x() } else if b { x() } else { x() }  ->  { x() }
//
// The recipe only fires when a final else clause is present (otherwise not all
// paths are covered) and every branch body prints identically.
type AllBranchesIdentical struct {
	recipe.Base
}

func (r *AllBranchesIdentical) Name() string {
	return "org.openrewrite.golang.codequality.AllBranchesIdentical"
}
func (r *AllBranchesIdentical) DisplayName() string {
	return "Remove if/else with identical branches"
}
func (r *AllBranchesIdentical) Description() string {
	return "Replace an if/else chain where every branch has the same body with just the body, " +
		"since the conditions have no effect on the outcome."
}
func (r *AllBranchesIdentical) Tags() []string {
	return []string{"cleanup", "redundancy", "RSPEC-S3923"}
}

func (r *AllBranchesIdentical) Editor() recipe.TreeVisitor {
	return visitor.Init(&allBranchesIdenticalVisitor{})
}

type allBranchesIdenticalVisitor struct {
	visitor.GoVisitor
}

func (v *allBranchesIdenticalVisitor) VisitIf(ifStmt *tree.If, p any) tree.J {
	ifStmt = v.GoVisitor.VisitIf(ifStmt, p).(*tree.If)

	if !allBranchBodiesIdentical(ifStmt) {
		return ifStmt
	}

	// Replace the entire if/else chain with just the then-body, preserving
	// the if statement's prefix (indentation).
	return ifStmt.Then.WithPrefix(ifStmt.Prefix)
}

// allBranchBodiesIdentical walks the if/else-if/else chain and returns true
// only when a final else clause exists and every branch body is identical.
func allBranchBodiesIdentical(ifStmt *tree.If) bool {
	reference := printBlockNormalized(ifStmt.Then)
	current := ifStmt

	for {
		if current.ElsePart == nil {
			// No final else -- not all paths are covered.
			return false
		}

		switch elseBody := current.ElsePart.Element.(type) {
		case *tree.If:
			if printBlockNormalized(elseBody.Then) != reference {
				return false
			}
			current = elseBody
		case *tree.Block:
			return printBlockNormalized(elseBody) == reference
		default:
			return false
		}
	}
}

func printBlockNormalized(block *tree.Block) string {
	return printer.Print(block.WithPrefix(tree.Space{}))
}
