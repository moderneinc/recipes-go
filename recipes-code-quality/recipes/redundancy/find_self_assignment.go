/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package redundancy

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindSelfAssignment finds `x = x` self-assignments which are always
// redundant and usually indicate a bug or leftover code.
// Staticcheck: SA4018
type FindSelfAssignment struct {
	recipe.Base
}

func (r *FindSelfAssignment) Name() string {
	return "org.openrewrite.golang.codequality.FindSelfAssignment"
}
func (r *FindSelfAssignment) DisplayName() string { return "Find self-assignment" }
func (r *FindSelfAssignment) Description() string {
	return "Find `x = x` self-assignments which are redundant and may indicate a bug."
}
func (r *FindSelfAssignment) Tags() []string { return []string{"cleanup", "redundancy", "lint"} }

func (r *FindSelfAssignment) Editor() recipe.TreeVisitor {
	return visitor.Init(&findSelfAssignmentVisitor{})
}

type findSelfAssignmentVisitor struct {
	visitor.GoVisitor
}

func (v *findSelfAssignmentVisitor) VisitAssignment(assign *tree.Assignment, p any) tree.J {
	assign = v.GoVisitor.VisitAssignment(assign, p).(*tree.Assignment)

	// Check if the left-hand side is an identifier.
	lhs, lhsOk := assign.Variable.(*tree.Identifier)
	if !lhsOk {
		return assign
	}

	// Check if the right-hand side is an identifier with the same name.
	rhs, rhsOk := assign.Value.Element.(*tree.Identifier)
	if !rhsOk {
		return assign
	}

	if lhs.Name != rhs.Name {
		return assign
	}

	assign = assign.WithMarkers(
		tree.FoundSearchResult(assign.Markers, "self-assignment is redundant"),
	)
	return assign
}
