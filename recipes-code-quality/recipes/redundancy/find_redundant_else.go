/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package redundancy

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindRedundantElse finds `if cond { ... return ... } else { ... }` patterns
// where the else block is unnecessary because the if block ends with a return
// statement, making the else redundant.
type FindRedundantElse struct {
	recipe.Base
}

func (r *FindRedundantElse) Name() string {
	return "org.openrewrite.golang.codequality.FindRedundantElse"
}
func (r *FindRedundantElse) DisplayName() string { return "Find redundant else after return" }
func (r *FindRedundantElse) Description() string {
	return "Find `if ... { return } else { ... }` where the else is redundant because the if block ends with a return."
}
func (r *FindRedundantElse) Tags() []string { return []string{"cleanup", "redundancy"} }

func (r *FindRedundantElse) Editor() recipe.TreeVisitor {
	return visitor.Init(&findRedundantElseVisitor{})
}

type findRedundantElseVisitor struct {
	visitor.GoVisitor
}

func (v *findRedundantElseVisitor) VisitIf(ifStmt *tree.If, p any) tree.J {
	ifStmt = v.GoVisitor.VisitIf(ifStmt, p).(*tree.If)

	// Must have an else clause.
	if ifStmt.ElsePart == nil {
		return ifStmt
	}

	// The then block must exist and have at least one statement.
	if ifStmt.Then == nil || len(ifStmt.Then.Statements) == 0 {
		return ifStmt
	}

	// Check if the last statement in the then block is a return.
	stmts := ifStmt.Then.Statements
	last := stmts[len(stmts)-1]
	if _, ok := last.Element.(*tree.Return); !ok {
		return ifStmt
	}

	ifStmt = ifStmt.WithMarkers(
		tree.FoundSearchResult(ifStmt.Markers, "else is redundant after a return in the if block"),
	)
	return ifStmt
}
