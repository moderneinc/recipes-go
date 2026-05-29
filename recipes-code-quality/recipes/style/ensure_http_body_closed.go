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

func (v *ensureHttpBodyClosedVisitor) VisitBlock(block *java.Block, p any) java.J {
	block = v.GoVisitor.VisitBlock(block, p).(*java.Block)

	var newStmts []java.RightPadded[java.Statement]
	changed := false

	for i, rp := range block.Statements {
		newStmts = append(newStmts, rp)

		if varName, ok := extractAssignedVar(rp.Element, isHttpBodyCall); ok {
			if hasDeferBodyCloseAfter(block.Statements, i, varName) {
				continue
			}
			deferStmt := buildDeferBodyClose(varName, rp.Element)
			newStmts = append(newStmts, java.RightPadded[java.Statement]{Element: deferStmt})
			changed = true
		}
	}

	if changed {
		return block.WithStatements(newStmts)
	}
	return block
}

// isHttpBodyCall returns true if the method invocation is http.Get/Post/Head or *.Do.
func isHttpBodyCall(mi *java.MethodInvocation) bool {
	if mi.Select == nil {
		return false
	}
	ident, ok := mi.Select.Element.(*java.Identifier)
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
func hasDeferBodyCloseAfter(stmts []java.RightPadded[java.Statement], i int, varName string) bool {
	for j := i + 1; j < len(stmts); j++ {
		d, ok := stmts[j].Element.(*golang.Defer)
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
func matchesDeferBodyClose(d *golang.Defer, varName string) bool {
	mi, ok := d.Expr.(*java.MethodInvocation)
	if !ok || mi.Name.Name != "Close" {
		return false
	}
	if mi.Select == nil {
		return false
	}
	// The select should be varName.Body (a FieldAccess)
	fa, ok := mi.Select.Element.(*java.FieldAccess)
	if !ok {
		return false
	}
	if fa.Name.Element.Name != "Body" {
		return false
	}
	ident, ok := fa.Target.(*java.Identifier)
	if !ok {
		return false
	}
	return ident.Name == varName
}

// buildDeferBodyClose builds `defer varName.Body.Close()`.
func buildDeferBodyClose(varName string, originalStmt java.Statement) *golang.Defer {
	prefix := stmtPrefix(originalStmt)

	respIdent := &java.Identifier{
		ID:   uuid.New(),
		Name: varName,
	}
	bodyAccess := &java.FieldAccess{
		ID:     uuid.New(),
		Target: respIdent,
		Name: java.LeftPadded[*java.Identifier]{
			Element: &java.Identifier{
				ID:   uuid.New(),
				Name: "Body",
			},
		},
	}
	closeIdent := &java.Identifier{
		ID:   uuid.New(),
		Name: "Close",
	}
	closeCall := &java.MethodInvocation{
		ID:     uuid.New(),
		Prefix: java.SingleSpace,
		Select: &java.RightPadded[java.Expression]{Element: bodyAccess},
		Name:   closeIdent,
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
