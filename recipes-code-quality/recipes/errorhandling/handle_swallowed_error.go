/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// HandleSwallowedError transforms `if err != nil { return }` — where the error is
// checked but the bare return swallows it — into `if err != nil { return err }`.
type HandleSwallowedError struct {
	recipe.Base
}

func (r *HandleSwallowedError) Name() string {
	return "org.openrewrite.golang.codequality.HandleSwallowedError"
}
func (r *HandleSwallowedError) DisplayName() string { return "Handle swallowed error" }
func (r *HandleSwallowedError) Description() string {
	return "Replace `if err != nil { return }` with `if err != nil { return err }` so the error is propagated."
}
func (r *HandleSwallowedError) Tags() []string { return []string{"error-handling", "lint"} }

func (r *HandleSwallowedError) Editor() recipe.TreeVisitor {
	return visitor.Init(&handleSwallowedErrorVisitor{})
}

type handleSwallowedErrorVisitor struct {
	visitor.GoVisitor
}

func (v *handleSwallowedErrorVisitor) VisitIf(ifStmt *java.If, p any) java.J {
	ifStmt = v.GoVisitor.VisitIf(ifStmt, p).(*java.If)

	if !isErrNotNil(ifStmt.Condition) {
		return ifStmt
	}

	if ifStmt.Then == nil {
		return ifStmt
	}

	stmts := realStatements(ifStmt.Then.Statements)
	if len(stmts) != 1 {
		return ifStmt
	}

	ret, ok := stmts[0].Element.(*java.Return)
	if !ok || len(ret.Expressions) > 0 {
		return ifStmt
	}

	// Replace bare return with return err
	errIdent := &java.Identifier{Prefix: java.Space{Whitespace: " "}, Name: "err"}
	newRet := &java.Return{
		ID: ret.ID, Prefix: ret.Prefix, Markers: ret.Markers,
		Expressions: []java.RightPadded[java.Expression]{{Element: errIdent}},
	}

	// Rebuild the Then block with the new return
	newStmts := make([]java.RightPadded[java.Statement], len(ifStmt.Then.Statements))
	copy(newStmts, ifStmt.Then.Statements)
	for i, s := range newStmts {
		if _, ok := s.Element.(*java.Return); ok {
			newStmts[i] = java.RightPadded[java.Statement]{Element: newRet, After: s.After}
			break
		}
	}
	newThen := ifStmt.Then.WithStatements(newStmts)
	return ifStmt.WithThen(newThen)
}

// isErrNotNil checks whether an expression is `err != nil`.
func isErrNotNil(expr java.Expression) bool {
	bin, ok := expr.(*java.Binary)
	if !ok || bin.Operator.Element != java.NotEqual {
		return false
	}
	leftIdent, leftOk := bin.Left.(*java.Identifier)
	rightIdent, rightOk := bin.Right.(*java.Identifier)
	if !leftOk || !rightOk {
		return false
	}
	return leftIdent.Name == "err" && rightIdent.Name == "nil"
}

// realStatements returns statements that are not *java.Empty.
func realStatements(stmts []java.RightPadded[java.Statement]) []java.RightPadded[java.Statement] {
	var out []java.RightPadded[java.Statement]
	for _, s := range stmts {
		if _, isEmpty := s.Element.(*java.Empty); !isEmpty {
			out = append(out, s)
		}
	}
	return out
}
