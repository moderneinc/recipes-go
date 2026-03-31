/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindDeeplyNestedErrorCheck finds `if err != nil` checks that are nested
// three or more levels deep. Deeply nested error checks are hard to follow
// and often indicate that the function should be refactored.
type FindDeeplyNestedErrorCheck struct {
	recipe.Base
}

func (r *FindDeeplyNestedErrorCheck) Name() string {
	return "org.openrewrite.golang.codequality.FindDeeplyNestedErrorCheck"
}
func (r *FindDeeplyNestedErrorCheck) DisplayName() string {
	return "Find deeply nested error checks"
}
func (r *FindDeeplyNestedErrorCheck) Description() string {
	return "Find `if err != nil` checks nested three or more levels deep. Consider refactoring to reduce nesting."
}
func (r *FindDeeplyNestedErrorCheck) Tags() []string { return []string{"style", "lint"} }

func (r *FindDeeplyNestedErrorCheck) Editor() recipe.TreeVisitor {
	return visitor.Init(&findDeeplyNestedErrorCheckVisitor{})
}

type findDeeplyNestedErrorCheckVisitor struct {
	visitor.GoVisitor
	ifDepth int
}

func (v *findDeeplyNestedErrorCheckVisitor) VisitIf(ifStmt *tree.If, p any) tree.J {
	v.ifDepth++
	ifStmt = v.GoVisitor.VisitIf(ifStmt, p).(*tree.If)
	v.ifDepth--

	if v.ifDepth+1 < 3 {
		return ifStmt
	}

	if !isErrNotNil(ifStmt.Condition) {
		return ifStmt
	}

	ifStmt = ifStmt.WithMarkers(
		tree.FoundSearchResult(ifStmt.Markers, "deeply nested error check"),
	)
	return ifStmt
}

// isErrNotNil returns true if the expression is `err != nil`.
func isErrNotNil(expr tree.Expression) bool {
	bin, ok := expr.(*tree.Binary)
	if !ok || bin.Operator.Element != tree.NotEqual {
		return false
	}

	leftIdent, leftOk := bin.Left.(*tree.Identifier)
	rightIdent, rightOk := bin.Right.(*tree.Identifier)
	if !leftOk || !rightOk {
		return false
	}
	return leftIdent.Name == "err" && rightIdent.Name == "nil"
}
