/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling

import (
	"github.com/moderneinc/recipes-go/recipes/internal/lstutil"
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

	if ifStmt.Condition == nil || !lstutil.IsErrNotNil(ifStmt.Condition.Tree.Element) {
		return ifStmt
	}

	thenBlock, ok := ifStmt.ThenPart.Element.(*java.Block)
	if !ok {
		return ifStmt
	}

	stmts := realStatements(thenBlock.Statements)
	if len(stmts) != 1 {
		return ifStmt
	}

	ret, ok := stmts[0].Element.(*java.Return)
	if !ok || ret.Expression != nil {
		return ifStmt
	}

	// Only rewrite when the enclosing function returns a single `error`, so the
	// bare `return` is valid (named result) and `return err` keeps the arity.
	if !enclosingReturnsSingleError(v.Cursor()) {
		return ifStmt
	}

	// Replace bare return with return err
	errIdent := &java.Identifier{Prefix: java.Space{Whitespace: " "}, Name: "err"}
	newRet := &java.Return{
		ID: ret.ID, Prefix: ret.Prefix, Markers: ret.Markers,
		Expression: errIdent,
	}

	// Rebuild the Then block with the new return
	newStmts := make([]java.RightPadded[java.Statement], len(thenBlock.Statements))
	copy(newStmts, thenBlock.Statements)
	for i, s := range newStmts {
		if _, ok := s.Element.(*java.Return); ok {
			newStmts[i] = java.RightPadded[java.Statement]{Element: newRet, After: s.After}
			break
		}
	}
	newThenPart := ifStmt.ThenPart
	newThenPart.Element = thenBlock.WithStatements(newStmts)
	return ifStmt.WithThenPart(newThenPart)
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
