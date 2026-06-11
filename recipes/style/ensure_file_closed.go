/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/google/uuid"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// osFileOpenMethods lists os package functions that open files.
var osFileOpenMethods = map[string]bool{
	"Open":     true,
	"Create":   true,
	"OpenFile": true,
}

// EnsureFileClosed finds calls to `os.Open()`, `os.Create()`, and
// `os.OpenFile()` and inserts `defer f.Close()` after the assignment.
type EnsureFileClosed struct {
	recipe.Base
}

func (r *EnsureFileClosed) Name() string {
	return "org.openrewrite.golang.codequality.EnsureFileClosed"
}
func (r *EnsureFileClosed) DisplayName() string { return "Ensure file closed" }
func (r *EnsureFileClosed) Description() string {
	return "Find calls to `os.Open`, `os.Create`, and `os.OpenFile`. Ensure the returned file is closed to avoid resource leaks."
}
func (r *EnsureFileClosed) Tags() []string { return []string{"style", "os"} }

func (r *EnsureFileClosed) Editor() recipe.TreeVisitor {
	return visitor.Init(&ensureFileClosedVisitor{})
}

type ensureFileClosedVisitor struct {
	visitor.GoVisitor
}

func (v *ensureFileClosedVisitor) VisitBlock(block *java.Block, p any) java.J {
	block = v.GoVisitor.VisitBlock(block, p).(*java.Block)

	var newStmts []java.RightPadded[java.Statement]
	changed := false

	for i, rp := range block.Statements {
		newStmts = append(newStmts, rp)

		if varName, ok := extractAssignedVar(rp.Element, isOsFileOpen); ok {
			if hasDeferAfter(block.Statements, i, varName, "Close") {
				continue
			}
			deferStmt := buildDeferMethodCall(varName, "Close", rp.Element)
			newStmts = append(newStmts, java.RightPadded[java.Statement]{Element: deferStmt})
			changed = true
		}
	}

	if changed {
		return block.WithStatements(newStmts)
	}
	return block
}

// isOsFileOpen returns true if the method invocation is os.Open, os.Create, or os.OpenFile.
func isOsFileOpen(mi *java.MethodInvocation) bool {
	if mi.Select == nil {
		return false
	}
	ident, ok := mi.Select.Element.(*java.Identifier)
	if !ok || ident.Name != "os" {
		return false
	}
	return osFileOpenMethods[mi.Name.Name]
}

// extractAssignedVar extracts the variable name from an assignment statement
// whose RHS matches the given predicate. Supports both Assignment (x := expr)
// and MultiAssignment (x, err := expr).
func extractAssignedVar(stmt java.Statement, matchRHS func(*java.MethodInvocation) bool) (string, bool) {
	switch s := stmt.(type) {
	case *java.Assignment:
		mi, ok := unwrapMethodInvocation(s.Value.Element)
		if !ok || !matchRHS(mi) {
			return "", false
		}
		if ident, ok := s.Variable.(*java.Identifier); ok {
			return ident.Name, true
		}
	case *golang.MultiAssignment:
		if len(s.Values) == 0 || len(s.Variables) == 0 {
			return "", false
		}
		mi, ok := unwrapMethodInvocation(s.Values[0].Element)
		if !ok || !matchRHS(mi) {
			return "", false
		}
		if ident, ok := s.Variables[0].Element.(*java.Identifier); ok {
			return ident.Name, true
		}
	}
	return "", false
}

// unwrapMethodInvocation extracts a *java.MethodInvocation from an expression.
func unwrapMethodInvocation(expr java.Expression) (*java.MethodInvocation, bool) {
	mi, ok := expr.(*java.MethodInvocation)
	return mi, ok
}

// hasDeferAfter checks if any statement after index i in the block is a defer
// calling varName.methodName().
func hasDeferAfter(stmts []java.RightPadded[java.Statement], i int, varName, methodName string) bool {
	for j := i + 1; j < len(stmts); j++ {
		d, ok := stmts[j].Element.(*golang.Defer)
		if !ok {
			continue
		}
		if matchesDeferCall(d, varName, methodName) {
			return true
		}
	}
	return false
}

// matchesDeferCall returns true if the defer calls varName.methodName().
func matchesDeferCall(d *golang.Defer, varName, methodName string) bool {
	mi, ok := d.Expr.(*java.MethodInvocation)
	if !ok {
		return false
	}
	if mi.Name.Name != methodName {
		return false
	}
	if mi.Select == nil {
		return false
	}
	ident, ok := mi.Select.Element.(*java.Identifier)
	if !ok {
		return false
	}
	return ident.Name == varName
}

// buildDeferMethodCall builds a `defer varName.methodName()` statement,
// copying the indentation prefix from the original statement.
func buildDeferMethodCall(varName, methodName string, originalStmt java.Statement) *golang.Defer {
	prefix := stmtPrefix(originalStmt)

	selectIdent := &java.Identifier{
		ID:   uuid.New(),
		Name: varName,
	}
	methodIdent := &java.Identifier{
		ID:   uuid.New(),
		Name: methodName,
	}
	closeCall := &java.MethodInvocation{
		ID:     uuid.New(),
		Prefix: java.SingleSpace,
		Select: &java.RightPadded[java.Expression]{Element: selectIdent},
		Name:   methodIdent,
		Arguments: java.Container[java.Expression]{
			Before: java.EmptySpace,
		},
	}
	return &golang.Defer{
		ID:     uuid.New(),
		Prefix: prefix,
		Expr:   closeCall,
	}
}

// stmtPrefix extracts the leading whitespace from a statement.
// In Go's AST the indentation lives on the first token of the statement,
// not on the statement node's own Prefix field.
func stmtPrefix(stmt java.Statement) java.Space {
	switch s := stmt.(type) {
	case *java.Assignment:
		if id, ok := s.Variable.(*java.Identifier); ok && id.Prefix.Whitespace != "" {
			return id.Prefix
		}
		return s.Prefix
	case *golang.MultiAssignment:
		if len(s.Variables) > 0 {
			if id, ok := s.Variables[0].Element.(*java.Identifier); ok && id.Prefix.Whitespace != "" {
				return id.Prefix
			}
		}
		return s.Prefix
	case *golang.Defer:
		return s.Prefix
	case *java.MethodInvocation:
		return s.Prefix
	default:
		return java.Space{Whitespace: "\n\t"}
	}
}
