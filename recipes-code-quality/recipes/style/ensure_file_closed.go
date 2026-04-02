/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/google/uuid"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree"
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

func (v *ensureFileClosedVisitor) VisitBlock(block *tree.Block, p any) tree.J {
	block = v.GoVisitor.VisitBlock(block, p).(*tree.Block)

	var newStmts []tree.RightPadded[tree.Statement]
	changed := false

	for i, rp := range block.Statements {
		newStmts = append(newStmts, rp)

		if varName, ok := extractAssignedVar(rp.Element, isOsFileOpen); ok {
			if hasDeferAfter(block.Statements, i, varName, "Close") {
				continue
			}
			deferStmt := buildDeferMethodCall(varName, "Close", rp.Element)
			newStmts = append(newStmts, tree.RightPadded[tree.Statement]{Element: deferStmt})
			changed = true
		}
	}

	if changed {
		return block.WithStatements(newStmts)
	}
	return block
}

// isOsFileOpen returns true if the method invocation is os.Open, os.Create, or os.OpenFile.
func isOsFileOpen(mi *tree.MethodInvocation) bool {
	if mi.Select == nil {
		return false
	}
	ident, ok := mi.Select.Element.(*tree.Identifier)
	if !ok || ident.Name != "os" {
		return false
	}
	return osFileOpenMethods[mi.Name.Name]
}

// extractAssignedVar extracts the variable name from an assignment statement
// whose RHS matches the given predicate. Supports both Assignment (x := expr)
// and MultiAssignment (x, err := expr).
func extractAssignedVar(stmt tree.Statement, matchRHS func(*tree.MethodInvocation) bool) (string, bool) {
	switch s := stmt.(type) {
	case *tree.Assignment:
		mi, ok := unwrapMethodInvocation(s.Value.Element)
		if !ok || !matchRHS(mi) {
			return "", false
		}
		if ident, ok := s.Variable.(*tree.Identifier); ok {
			return ident.Name, true
		}
	case *tree.MultiAssignment:
		if len(s.Values) == 0 || len(s.Variables) == 0 {
			return "", false
		}
		mi, ok := unwrapMethodInvocation(s.Values[0].Element)
		if !ok || !matchRHS(mi) {
			return "", false
		}
		if ident, ok := s.Variables[0].Element.(*tree.Identifier); ok {
			return ident.Name, true
		}
	}
	return "", false
}

// unwrapMethodInvocation extracts a *tree.MethodInvocation from an expression.
func unwrapMethodInvocation(expr tree.Expression) (*tree.MethodInvocation, bool) {
	mi, ok := expr.(*tree.MethodInvocation)
	return mi, ok
}

// hasDeferAfter checks if any statement after index i in the block is a defer
// calling varName.methodName().
func hasDeferAfter(stmts []tree.RightPadded[tree.Statement], i int, varName, methodName string) bool {
	for j := i + 1; j < len(stmts); j++ {
		d, ok := stmts[j].Element.(*tree.Defer)
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
func matchesDeferCall(d *tree.Defer, varName, methodName string) bool {
	mi, ok := d.Expr.(*tree.MethodInvocation)
	if !ok {
		return false
	}
	if mi.Name.Name != methodName {
		return false
	}
	if mi.Select == nil {
		return false
	}
	ident, ok := mi.Select.Element.(*tree.Identifier)
	if !ok {
		return false
	}
	return ident.Name == varName
}

// buildDeferMethodCall builds a `defer varName.methodName()` statement,
// copying the indentation prefix from the original statement.
func buildDeferMethodCall(varName, methodName string, originalStmt tree.Statement) *tree.Defer {
	prefix := stmtPrefix(originalStmt)

	selectIdent := &tree.Identifier{
		ID:   uuid.New(),
		Name: varName,
	}
	methodIdent := &tree.Identifier{
		ID:   uuid.New(),
		Name: methodName,
	}
	closeCall := &tree.MethodInvocation{
		ID:     uuid.New(),
		Prefix: tree.SingleSpace,
		Select: &tree.RightPadded[tree.Expression]{Element: selectIdent},
		Name:   methodIdent,
		Arguments: tree.Container[tree.Expression]{
			Before: tree.EmptySpace,
		},
	}
	return &tree.Defer{
		ID:     uuid.New(),
		Prefix: prefix,
		Expr:   closeCall,
	}
}

// stmtPrefix extracts the leading whitespace from a statement.
// In Go's AST the indentation lives on the first token of the statement,
// not on the statement node's own Prefix field.
func stmtPrefix(stmt tree.Statement) tree.Space {
	switch s := stmt.(type) {
	case *tree.Assignment:
		if id, ok := s.Variable.(*tree.Identifier); ok && id.Prefix.Whitespace != "" {
			return id.Prefix
		}
		return s.Prefix
	case *tree.MultiAssignment:
		if len(s.Variables) > 0 {
			if id, ok := s.Variables[0].Element.(*tree.Identifier); ok && id.Prefix.Whitespace != "" {
				return id.Prefix
			}
		}
		return s.Prefix
	case *tree.Defer:
		return s.Prefix
	case *tree.MethodInvocation:
		return s.Prefix
	default:
		return tree.Space{Whitespace: "\n\t"}
	}
}
