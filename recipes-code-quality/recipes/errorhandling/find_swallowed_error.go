/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindSwallowedError finds `if err != nil { return }` — where the error is
// checked but the bare return swallows it instead of propagating or handling it.
type FindSwallowedError struct {
	recipe.Base
}

func (r *FindSwallowedError) Name() string {
	return "org.openrewrite.golang.codequality.FindSwallowedError"
}
func (r *FindSwallowedError) DisplayName() string { return "Find swallowed error" }
func (r *FindSwallowedError) Description() string {
	return "Find `if err != nil { return }` where the error is checked but swallowed by a bare return."
}
func (r *FindSwallowedError) Tags() []string { return []string{"error-handling", "lint"} }

func (r *FindSwallowedError) Editor() recipe.TreeVisitor {
	return visitor.Init(&findSwallowedErrorVisitor{})
}

type findSwallowedErrorVisitor struct {
	visitor.GoVisitor
}

func (v *findSwallowedErrorVisitor) VisitIf(ifStmt *tree.If, p any) tree.J {
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

	// Check if the then block has exactly 1 real statement which is a bare return.
	if ifStmt.Then == nil {
		return ifStmt
	}

	stmts := realStatements(ifStmt.Then.Statements)
	if len(stmts) != 1 {
		return ifStmt
	}

	ret, ok := stmts[0].Element.(*tree.Return)
	if !ok {
		return ifStmt
	}

	// A bare return has 0 expressions.
	if len(ret.Expressions) > 0 {
		return ifStmt
	}

	ifStmt = ifStmt.WithMarkers(
		tree.FoundSearchResult(ifStmt.Markers, "error checked but swallowed by bare return"),
	)
	return ifStmt
}

// realStatements filters out Empty sentinel elements from a statement list.
func realStatements(stmts []tree.RightPadded[tree.Statement]) []tree.RightPadded[tree.Statement] {
	var result []tree.RightPadded[tree.Statement]
	for _, s := range stmts {
		if _, isEmpty := s.Element.(*tree.Empty); !isEmpty {
			result = append(result, s)
		}
	}
	return result
}
