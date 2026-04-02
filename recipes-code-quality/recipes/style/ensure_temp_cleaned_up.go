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

func (v *ensureTempCleanedUpVisitor) VisitBlock(block *tree.Block, p any) tree.J {
	block = v.GoVisitor.VisitBlock(block, p).(*tree.Block)

	var newStmts []tree.RightPadded[tree.Statement]
	changed := false

	for i, rp := range block.Statements {
		newStmts = append(newStmts, rp)

		if varName, ok := extractAssignedVar(rp.Element, isOsCreateTemp); ok {
			if hasDeferRemoveAfter(block.Statements, i, varName) {
				continue
			}
			deferStmt := buildDeferOsRemove(varName, rp.Element)
			newStmts = append(newStmts, tree.RightPadded[tree.Statement]{Element: deferStmt})
			changed = true
		}
	}

	if changed {
		return block.WithStatements(newStmts)
	}
	return block
}

// isOsCreateTemp returns true if the method invocation is os.CreateTemp.
func isOsCreateTemp(mi *tree.MethodInvocation) bool {
	if mi.Select == nil {
		return false
	}
	ident, ok := mi.Select.Element.(*tree.Identifier)
	if !ok || ident.Name != "os" {
		return false
	}
	return mi.Name.Name == "CreateTemp"
}

// hasDeferRemoveAfter checks if any statement after index i is a defer
// calling os.Remove(varName.Name()).
func hasDeferRemoveAfter(stmts []tree.RightPadded[tree.Statement], i int, varName string) bool {
	for j := i + 1; j < len(stmts); j++ {
		d, ok := stmts[j].Element.(*tree.Defer)
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
func matchesDeferOsRemove(d *tree.Defer, varName string) bool {
	mi, ok := d.Expr.(*tree.MethodInvocation)
	if !ok || mi.Name.Name != "Remove" {
		return false
	}
	if mi.Select == nil {
		return false
	}
	ident, ok := mi.Select.Element.(*tree.Identifier)
	if !ok || ident.Name != "os" {
		return false
	}
	// Check that the argument is varName.Name()
	if len(mi.Arguments.Elements) != 1 {
		return false
	}
	argMi, ok := mi.Arguments.Elements[0].Element.(*tree.MethodInvocation)
	if !ok || argMi.Name.Name != "Name" {
		return false
	}
	if argMi.Select == nil {
		return false
	}
	argIdent, ok := argMi.Select.Element.(*tree.Identifier)
	if !ok {
		return false
	}
	return argIdent.Name == varName
}

// buildDeferOsRemove builds `defer os.Remove(varName.Name())`.
func buildDeferOsRemove(varName string, originalStmt tree.Statement) *tree.Defer {
	prefix := stmtPrefix(originalStmt)

	// Build varName.Name()
	nameCall := &tree.MethodInvocation{
		ID:     uuid.New(),
		Select: &tree.RightPadded[tree.Expression]{Element: &tree.Identifier{ID: uuid.New(), Name: varName}},
		Name:   &tree.Identifier{ID: uuid.New(), Name: "Name"},
		Arguments: tree.Container[tree.Expression]{
			Before: tree.EmptySpace,
		},
	}

	// Build os.Remove(varName.Name())
	removeCall := &tree.MethodInvocation{
		ID:     uuid.New(),
		Prefix: tree.SingleSpace,
		Select: &tree.RightPadded[tree.Expression]{Element: &tree.Identifier{ID: uuid.New(), Name: "os"}},
		Name:   &tree.Identifier{ID: uuid.New(), Name: "Remove"},
		Arguments: tree.Container[tree.Expression]{
			Before: tree.EmptySpace,
			Elements: []tree.RightPadded[tree.Expression]{
				{Element: nameCall},
			},
		},
	}
	return &tree.Defer{
		ID:     uuid.New(),
		Prefix: prefix,
		Expr:   removeCall,
	}
}
