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

// EnsureHttpBodyClosed finds calls to `http.Get`, `http.Post`, `http.Head`,
// and `client.Do` and inserts `defer resp.Body.Close()` after the assignment.
type EnsureHttpBodyClosed struct {
	recipe.Base
}

func (r *EnsureHttpBodyClosed) Name() string {
	return "org.openrewrite.golang.codequality.EnsureHttpBodyClosed"
}
func (r *EnsureHttpBodyClosed) DisplayName() string { return "Ensure HTTP body closed" }
func (r *EnsureHttpBodyClosed) Description() string {
	return "Find calls to `http.Get`, `http.Post`, `http.Head`, and `client.Do` whose response body must be closed to avoid resource leaks."
}
func (r *EnsureHttpBodyClosed) Tags() []string { return []string{"style", "resource-management"} }

func (r *EnsureHttpBodyClosed) Editor() recipe.TreeVisitor {
	return visitor.Init(&ensureHttpBodyClosedVisitor{})
}

type ensureHttpBodyClosedVisitor struct {
	visitor.GoVisitor
}

// httpBodyMethods lists the net/http convenience functions whose response
// body must always be closed by the caller.
var httpBodyMethods = map[string]bool{
	"Get":  true,
	"Post": true,
	"Head": true,
}

func (v *ensureHttpBodyClosedVisitor) VisitBlock(block *tree.Block, p any) tree.J {
	block = v.GoVisitor.VisitBlock(block, p).(*tree.Block)

	var newStmts []tree.RightPadded[tree.Statement]
	changed := false

	for i, rp := range block.Statements {
		newStmts = append(newStmts, rp)

		if varName, ok := extractAssignedVar(rp.Element, isHttpBodyCall); ok {
			if hasDeferBodyCloseAfter(block.Statements, i, varName) {
				continue
			}
			deferStmt := buildDeferBodyClose(varName, rp.Element)
			newStmts = append(newStmts, tree.RightPadded[tree.Statement]{Element: deferStmt})
			changed = true
		}
	}

	if changed {
		return block.WithStatements(newStmts)
	}
	return block
}

// isHttpBodyCall returns true if the method invocation is http.Get/Post/Head or *.Do.
func isHttpBodyCall(mi *tree.MethodInvocation) bool {
	if mi.Select == nil {
		return false
	}
	ident, ok := mi.Select.Element.(*tree.Identifier)
	if !ok {
		return false
	}
	// Match http.Get / http.Post / http.Head
	if ident.Name == "http" && httpBodyMethods[mi.Name.Name] {
		return true
	}
	// Match client.Do (any receiver calling Do)
	if mi.Name.Name == "Do" {
		return true
	}
	return false
}

// hasDeferBodyCloseAfter checks if any statement after index i is
// defer varName.Body.Close().
func hasDeferBodyCloseAfter(stmts []tree.RightPadded[tree.Statement], i int, varName string) bool {
	for j := i + 1; j < len(stmts); j++ {
		d, ok := stmts[j].Element.(*tree.Defer)
		if !ok {
			continue
		}
		if matchesDeferBodyClose(d, varName) {
			return true
		}
	}
	return false
}

// matchesDeferBodyClose returns true if the defer calls varName.Body.Close().
func matchesDeferBodyClose(d *tree.Defer, varName string) bool {
	mi, ok := d.Expr.(*tree.MethodInvocation)
	if !ok || mi.Name.Name != "Close" {
		return false
	}
	if mi.Select == nil {
		return false
	}
	// The select should be varName.Body (a FieldAccess)
	fa, ok := mi.Select.Element.(*tree.FieldAccess)
	if !ok {
		return false
	}
	if fa.Name.Element.Name != "Body" {
		return false
	}
	ident, ok := fa.Target.(*tree.Identifier)
	if !ok {
		return false
	}
	return ident.Name == varName
}

// buildDeferBodyClose builds `defer varName.Body.Close()`.
func buildDeferBodyClose(varName string, originalStmt tree.Statement) *tree.Defer {
	prefix := stmtPrefix(originalStmt)

	respIdent := &tree.Identifier{
		ID:   uuid.New(),
		Name: varName,
	}
	bodyAccess := &tree.FieldAccess{
		ID:     uuid.New(),
		Target: respIdent,
		Name: tree.LeftPadded[*tree.Identifier]{
			Element: &tree.Identifier{
				ID:   uuid.New(),
				Name: "Body",
			},
		},
	}
	closeIdent := &tree.Identifier{
		ID:   uuid.New(),
		Name: "Close",
	}
	closeCall := &tree.MethodInvocation{
		ID:     uuid.New(),
		Prefix: tree.SingleSpace,
		Select: &tree.RightPadded[tree.Expression]{Element: bodyAccess},
		Name:   closeIdent,
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
