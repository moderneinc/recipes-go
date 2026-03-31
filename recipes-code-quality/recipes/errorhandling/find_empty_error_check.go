/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindEmptyErrorCheck finds `if err != nil { }` blocks with empty bodies where
// an error is checked but not handled.
type FindEmptyErrorCheck struct {
	recipe.Base
}

func (r *FindEmptyErrorCheck) Name() string {
	return "org.openrewrite.golang.codequality.FindEmptyErrorCheck"
}
func (r *FindEmptyErrorCheck) DisplayName() string { return "Find empty error check" }
func (r *FindEmptyErrorCheck) Description() string {
	return "Find `if err != nil { }` blocks with empty bodies where the error is checked but not handled."
}
func (r *FindEmptyErrorCheck) Tags() []string { return []string{"errorhandling", "lint"} }

func (r *FindEmptyErrorCheck) Editor() recipe.TreeVisitor {
	return visitor.Init(&findEmptyErrorCheckVisitor{})
}

type findEmptyErrorCheckVisitor struct {
	visitor.GoVisitor
}

func (v *findEmptyErrorCheckVisitor) VisitIf(ifStmt *tree.If, p any) tree.J {
	ifStmt = v.GoVisitor.VisitIf(ifStmt, p).(*tree.If)

	// Check if the condition is `err != nil`.
	bin, ok := ifStmt.Condition.(*tree.Binary)
	if !ok || bin.Operator.Element != tree.NotEqual {
		return ifStmt
	}

	leftIdent, leftOk := bin.Left.(*tree.Identifier)
	rightIdent, rightOk := bin.Right.(*tree.Identifier)
	if !leftOk || !rightOk {
		return ifStmt
	}
	if leftIdent.Name != "err" || rightIdent.Name != "nil" {
		return ifStmt
	}

	// Check if the then block is empty (no real statements).
	if ifStmt.Then == nil {
		return ifStmt
	}
	if countRealStatements(ifStmt.Then.Statements) > 0 {
		return ifStmt
	}

	ifStmt = ifStmt.WithMarkers(
		tree.FoundSearchResult(ifStmt.Markers, "error checked but not handled"),
	)
	return ifStmt
}

// countRealStatements counts statements in a block, excluding Empty sentinels.
func countRealStatements(stmts []tree.RightPadded[tree.Statement]) int {
	count := 0
	for _, s := range stmts {
		if _, isEmpty := s.Element.(*tree.Empty); !isEmpty {
			count++
		}
	}
	return count
}
