/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// HandleCheckedError finds `if err != nil { }` blocks with empty bodies where
// an error is checked but not handled, and fills them with `return err`.
type HandleCheckedError struct {
	recipe.Base
}

func (r *HandleCheckedError) Name() string {
	return "org.openrewrite.golang.codequality.HandleCheckedError"
}
func (r *HandleCheckedError) DisplayName() string { return "Handle checked error" }
func (r *HandleCheckedError) Description() string {
	return "Replace `if err != nil { }` with `if err != nil { return err }` so the error is propagated."
}
func (r *HandleCheckedError) Tags() []string { return []string{"errorhandling", "lint"} }

func (r *HandleCheckedError) Editor() recipe.TreeVisitor {
	return visitor.Init(&handleCheckedErrorVisitor{})
}

type handleCheckedErrorVisitor struct {
	visitor.GoVisitor
}

func (v *handleCheckedErrorVisitor) VisitIf(ifStmt *tree.If, p any) tree.J {
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

	// Derive indentation from the block's End space. End.Whitespace is
	// the whitespace before `}`, e.g. "\n\t". The return statement sits
	// one indent level deeper.
	endWS := ifStmt.Then.End.Whitespace
	returnPrefix := tree.Space{Whitespace: endWS + "\t"}

	errIdent := &tree.Identifier{Prefix: tree.Space{Whitespace: " "}, Name: "err"}
	returnStmt := &tree.Return{
		Prefix:      returnPrefix,
		Expressions: []tree.RightPadded[tree.Expression]{{Element: errIdent}},
	}

	newStmts := []tree.RightPadded[tree.Statement]{
		{Element: returnStmt},
	}
	newThen := ifStmt.Then.WithStatements(newStmts)
	// Keep the closing `}` at its original indent level.
	newThen = newThen.WithEnd(ifStmt.Then.End)
	return ifStmt.WithThen(newThen)
}

// countRealStatements counts statements that are not *tree.Empty.
func countRealStatements(stmts []tree.RightPadded[tree.Statement]) int {
	n := 0
	for _, s := range stmts {
		if _, isEmpty := s.Element.(*tree.Empty); !isEmpty {
			n++
		}
	}
	return n
}
