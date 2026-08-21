/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

// Package lstutil holds small helpers shared across recipe packages for
// working with the LST and the visitor cursor.
package lstutil

import (
	"strings"

	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// Reports whether the block at the cursor is a function's body (its parent is a
// function declaration) rather than a nested block such as a loop or if body.
func IsFunctionBodyBlock(c *visitor.Cursor) bool {
	parent := c.Parent()
	if parent == nil {
		return false
	}
	switch parent.Value().(type) {
	case *java.MethodDeclaration, *golang.MethodDeclaration:
		return true
	}
	return false
}

// Returns the indentation (text after the last newline) of a Space. When the
// Space holds comments, that indentation sits in the last comment's suffix.
func BaseIndent(space java.Space) string {
	ws := space.Whitespace
	if n := len(space.Comments); n > 0 {
		ws = space.Comments[n-1].Suffix
	}
	if idx := strings.LastIndex(ws, "\n"); idx >= 0 {
		return ws[idx+1:]
	}
	return ws
}

// Returns a prefix that starts a new line at the indentation of space, dropping
// its comments: a comment belongs to the statement it was written above, so a
// synthesized statement must not carry it.
func IndentPrefix(space java.Space) java.Space {
	return java.Space{Whitespace: "\n" + BaseIndent(space)}
}

// Reports whether the If at the cursor is the inner statement of a
// golang.StatementWithInit, i.e. it carried an `if init; cond` init clause.
func IsInitWrappedIf(c *visitor.Cursor) bool {
	parent := c.Parent()
	if parent == nil {
		return false
	}
	_, ok := parent.Value().(*golang.StatementWithInit)
	return ok
}

// Reports whether an expression is `err != nil`.
func IsErrNotNil(expr java.Expression) bool {
	return IsNotNilCheck(expr, "err")
}

// Reports whether an expression is `<name> != nil`.
func IsNotNilCheck(expr java.Expression, name string) bool {
	bin, ok := expr.(*java.Binary)
	if !ok || bin.Operator.Element != java.NotEqual {
		return false
	}
	leftIdent, leftOk := bin.Left.(*java.Identifier)
	rightIdent, rightOk := bin.Right.(*java.Identifier)
	if !leftOk || !rightOk {
		return false
	}
	return leftIdent.Name == name && rightIdent.Name == "nil"
}
