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

// EnsureTempCleanedUp finds calls to `os.CreateTemp` and inserts
// `defer os.Remove(f.Name())` after the assignment.
type EnsureTempCleanedUp struct {
	recipe.Base
}

func (r *EnsureTempCleanedUp) Name() string {
	return "org.openrewrite.golang.codequality.EnsureTempCleanedUp"
}
func (r *EnsureTempCleanedUp) DisplayName() string { return "Ensure temp cleaned up" }
func (r *EnsureTempCleanedUp) Description() string {
	return "Find calls to `os.CreateTemp`. Temporary files should be cleaned up when no longer needed."
}
func (r *EnsureTempCleanedUp) Tags() []string { return []string{"style", "resource-management"} }

func (r *EnsureTempCleanedUp) Editor() recipe.TreeVisitor {
	return visitor.Init(&ensureTempCleanedUpVisitor{})
}

type ensureTempCleanedUpVisitor struct {
	visitor.GoVisitor
}

func (v *ensureTempCleanedUpVisitor) VisitBlock(block *java.Block, p any) java.J {
	block = v.GoVisitor.VisitBlock(block, p).(*java.Block)

	var newStmts []java.RightPadded[java.Statement]
	changed := false

	for i, rp := range block.Statements {
		newStmts = append(newStmts, rp)

		if varName, ok := extractAssignedVar(rp.Element, isOsCreateTemp); ok {
			if hasDeferRemoveAfter(block.Statements, i, varName) {
				continue
			}
			deferStmt := buildDeferOsRemove(varName, rp.Element)
			newStmts = append(newStmts, java.RightPadded[java.Statement]{Element: deferStmt})
			changed = true
		}
	}

	if changed {
		return block.WithStatements(newStmts)
	}
	return block
}

// isOsCreateTemp returns true if the method invocation is os.CreateTemp.
func isOsCreateTemp(mi *java.MethodInvocation) bool {
	if mi.Select == nil {
		return false
	}
	ident, ok := mi.Select.Element.(*java.Identifier)
	if !ok || ident.Name != "os" {
		return false
	}
	return mi.Name.Name == "CreateTemp"
}

// hasDeferRemoveAfter checks if any statement after index i is a defer
// calling os.Remove(varName.Name()).
func hasDeferRemoveAfter(stmts []java.RightPadded[java.Statement], i int, varName string) bool {
	for j := i + 1; j < len(stmts); j++ {
		d, ok := stmts[j].Element.(*golang.Defer)
		if !ok {
			continue
		}
		if matchesDeferOsRemove(d, varName) {
			return true
		}
	}
	return false
}

// matchesDeferOsRemove returns true if the defer calls os.Remove(varName.Name()).
func matchesDeferOsRemove(d *golang.Defer, varName string) bool {
	mi, ok := d.Expr.(*java.MethodInvocation)
	if !ok || mi.Name.Name != "Remove" {
		return false
	}
	if mi.Select == nil {
		return false
	}
	ident, ok := mi.Select.Element.(*java.Identifier)
	if !ok || ident.Name != "os" {
		return false
	}
	// Check that the argument is varName.Name()
	if len(mi.Arguments.Elements) != 1 {
		return false
	}
	argMi, ok := mi.Arguments.Elements[0].Element.(*java.MethodInvocation)
	if !ok || argMi.Name.Name != "Name" {
		return false
	}
	if argMi.Select == nil {
		return false
	}
	argIdent, ok := argMi.Select.Element.(*java.Identifier)
	if !ok {
		return false
	}
	return argIdent.Name == varName
}

// buildDeferOsRemove builds `defer os.Remove(varName.Name())`.
func buildDeferOsRemove(varName string, originalStmt java.Statement) *golang.Defer {
	prefix := stmtPrefix(originalStmt)

	// Build varName.Name()
	nameCall := &java.MethodInvocation{
		ID:     uuid.New(),
		Select: &java.RightPadded[java.Expression]{Element: &java.Identifier{ID: uuid.New(), Name: varName}},
		Name:   &java.Identifier{ID: uuid.New(), Name: "Name"},
		Arguments: java.Container[java.Expression]{
			Before: java.EmptySpace,
		},
	}

	// Build os.Remove(varName.Name())
	removeCall := &java.MethodInvocation{
		ID:     uuid.New(),
		Prefix: java.SingleSpace,
		Select: &java.RightPadded[java.Expression]{Element: &java.Identifier{ID: uuid.New(), Name: "os"}},
		Name:   &java.Identifier{ID: uuid.New(), Name: "Remove"},
		Arguments: java.Container[java.Expression]{
			Before: java.EmptySpace,
			Elements: []java.RightPadded[java.Expression]{
				{Element: nameCall},
			},
		},
	}
	return &golang.Defer{
		ID:     uuid.New(),
		Prefix: prefix,
		Expr:   removeCall,
	}
}
