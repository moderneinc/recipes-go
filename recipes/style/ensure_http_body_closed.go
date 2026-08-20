/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/google/uuid"
	"github.com/moderneinc/recipes-go/diagnostic"
	"github.com/moderneinc/recipes-go/recipes/internal/lstutil"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// EnsureHttpBodyClosed finds assignments of a `*http.Response` and inserts
// `defer resp.Body.Close()` after the assignment.
type EnsureHttpBodyClosed struct {
	recipe.Base
}

func (r *EnsureHttpBodyClosed) Name() string {
	return "org.openrewrite.golang.codequality.EnsureHttpBodyClosed"
}
func (r *EnsureHttpBodyClosed) DisplayName() string { return "Ensure HTTP body closed" }
func (r *EnsureHttpBodyClosed) Description() string {
	return "Find assignments of a `*http.Response`, as returned by `http.Get`, `http.Post`, `http.Head` or `client.Do`. Its body must be closed to avoid resource leaks."
}
func (r *EnsureHttpBodyClosed) Tags() []string { return []string{"style", "resource-management"} }

func (r *EnsureHttpBodyClosed) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{
		{DiagnosticID: "bodyclose", Tool: diagnostic.GolangciLint, HasFix: false},
	}
}

func (r *EnsureHttpBodyClosed) Editor() recipe.TreeVisitor {
	return visitor.Init(&ensureHttpBodyClosedVisitor{})
}

type ensureHttpBodyClosedVisitor struct {
	visitor.GoVisitor
}

func (v *ensureHttpBodyClosedVisitor) VisitBlock(block *java.Block, p any) java.J {
	block = v.GoVisitor.VisitBlock(block, p).(*java.Block)
	return insertDefer(block, isHttpResponse, hasDeferBodyCloseAfter,
		func(a acquisition, acquire java.Statement) *golang.Defer {
			return buildDeferBodyClose(a, acquire)
		})
}

func isHttpResponse(a acquisition) bool {
	return typeIs(a.varType, "net/http.Response")
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

// buildDeferBodyClose builds `defer varName.Body.Close()`. The response type
// resolves the Body field, and Body's own type the Close method.
func buildDeferBodyClose(a acquisition, originalStmt java.Statement) *golang.Defer {
	prefix := stmtPrefix(originalStmt)

	respIdent := &java.Identifier{
		ID:   uuid.New(),
		Name: a.varName,
		Type: a.varType,
	}
	bodyType := lstutil.FieldOn(a.varType, "Body")
	bodyAccess := &java.FieldAccess{
		ID:     uuid.New(),
		Target: respIdent,
		Name: java.LeftPadded[*java.Identifier]{
			Element: &java.Identifier{
				ID:   uuid.New(),
				Name: "Body",
				Type: bodyType,
			},
		},
		Type: bodyType,
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
		MethodType: lstutil.MethodOn(bodyType, "Close"),
	}
	return &golang.Defer{
		ID:     uuid.New(),
		Prefix: prefix,
		Expr:   closeCall,
	}
}
