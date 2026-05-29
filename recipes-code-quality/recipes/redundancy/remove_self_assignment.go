/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package redundancy

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// RemoveSelfAssignment removes `x = x` self-assignments which are always
// redundant and usually indicate a bug or leftover code.
// Staticcheck: SA4018
type RemoveSelfAssignment struct {
	recipe.Base
}

func (r *RemoveSelfAssignment) Name() string {
	return "org.openrewrite.golang.codequality.RemoveSelfAssignment"
}
func (r *RemoveSelfAssignment) DisplayName() string { return "Remove self-assignment" }
func (r *RemoveSelfAssignment) Description() string {
	return "Remove `x = x` self-assignments which are redundant and may indicate a bug."
}
func (r *RemoveSelfAssignment) Tags() []string { return []string{"cleanup", "redundancy", "lint"} }

func (r *RemoveSelfAssignment) Editor() recipe.TreeVisitor {
	return visitor.Init(&removeSelfAssignmentVisitor{})
}

type removeSelfAssignmentVisitor struct {
	visitor.GoVisitor
}

func (v *removeSelfAssignmentVisitor) VisitAssignment(assign *java.Assignment, p any) java.J {
	assign = v.GoVisitor.VisitAssignment(assign, p).(*java.Assignment)

	// Check if the left-hand side is an identifier.
	lhs, lhsOk := assign.Variable.(*java.Identifier)
	if !lhsOk {
		return assign
	}

	// Check if the right-hand side is an identifier with the same name.
	rhs, rhsOk := assign.Value.Element.(*java.Identifier)
	if !rhsOk {
		return assign
	}

	if lhs.Name != rhs.Name {
		return assign
	}

	// Replace the self-assignment with an empty statement.
	return &java.Empty{Prefix: assign.Variable.(*java.Identifier).Prefix}
}
