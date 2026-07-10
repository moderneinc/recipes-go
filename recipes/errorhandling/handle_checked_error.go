/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
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

func (v *handleCheckedErrorVisitor) VisitIf(ifStmt *java.If, p any) java.J {
	ifStmt = v.GoVisitor.VisitIf(ifStmt, p).(*java.If)

	// Check if the condition is `err != nil`.
	if ifStmt.Condition == nil {
		return ifStmt
	}
	bin, ok := ifStmt.Condition.Tree.Element.(*java.Binary)
	if !ok || bin.Operator.Element != java.NotEqual {
		return ifStmt
	}

	leftIdent, leftOk := bin.Left.(*java.Identifier)
	rightIdent, rightOk := bin.Right.(*java.Identifier)
	if !leftOk || !rightOk {
		return ifStmt
	}
	if leftIdent.Name != "err" || rightIdent.Name != "nil" {
		return ifStmt
	}

	// Check if the then block is empty (no real statements).
	thenBlock, ok := ifStmt.ThenPart.Element.(*java.Block)
	if !ok {
		return ifStmt
	}
	if countRealStatements(thenBlock.Statements) > 0 {
		return ifStmt
	}

	// Only propagate when the enclosing function returns a single `error`;
	// otherwise `return err` would not compile (wrong arity, or nothing to return).
	if !enclosingReturnsSingleError(v.Cursor()) {
		return ifStmt
	}

	// Derive indentation from the block's End space. End.Whitespace is
	// the whitespace before `}`, e.g. "\n\t". The return statement sits
	// one indent level deeper.
	endWS := thenBlock.End.Whitespace
	returnPrefix := java.Space{Whitespace: endWS + "\t"}

	errIdent := &java.Identifier{Prefix: java.Space{Whitespace: " "}, Name: "err"}
	returnStmt := &java.Return{
		Prefix:     returnPrefix,
		Expression: errIdent,
	}

	newStmts := []java.RightPadded[java.Statement]{
		{Element: returnStmt},
	}
	newThen := thenBlock.WithStatements(newStmts)
	// Keep the closing `}` at its original indent level.
	newThen = newThen.WithEnd(thenBlock.End)
	newThenPart := ifStmt.ThenPart
	newThenPart.Element = newThen
	return ifStmt.WithThenPart(newThenPart)
}

// enclosingReturnsSingleError reports whether the function (or function literal)
// enclosing the cursor declares exactly one result of type `error`. A bare
// `return err` can only be synthesized soundly in that case — a void function
// has nothing to return, and a multi-value result would be the wrong arity.
func enclosingReturnsSingleError(c *visitor.Cursor) bool {
	md, ok := visitor.FirstEnclosing[*java.MethodDeclaration](c)
	if !ok || md.ReturnType == nil {
		return false
	}
	switch rt := md.ReturnType.(type) {
	case *java.Identifier:
		// Unnamed single result, e.g. `func f() error`.
		return rt.Name == "error"
	case *golang.TypeList:
		// Parenthesized result list, e.g. `func f() (err error)`.
		if len(rt.Types.Elements) != 1 {
			return false
		}
		return isErrorResult(rt.Types.Elements[0].Element)
	}
	return false
}

// isErrorResult reports whether a single entry in a parenthesized result list
// denotes the `error` type. A named result (`err error`) is represented as a
// VariableDeclarations whose type expression is the `error` identifier.
func isErrorResult(stmt java.Statement) bool {
	if vd, ok := stmt.(*java.VariableDeclarations); ok {
		ident, ok := vd.TypeExpr.(*java.Identifier)
		return ok && ident.Name == "error"
	}
	return false
}

// countRealStatements counts statements that are not *java.Empty.
func countRealStatements(stmts []java.RightPadded[java.Statement]) int {
	n := 0
	for _, s := range stmts {
		if _, isEmpty := s.Element.(*java.Empty); !isEmpty {
			n++
		}
	}
	return n
}
