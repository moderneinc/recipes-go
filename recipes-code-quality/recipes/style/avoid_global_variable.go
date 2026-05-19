/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/moderneinc/recipes-go/recipes-code-quality/diagnostic"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// AvoidGlobalVariable finds package-level `var` declarations (mutable global state).
// golangci-lint: gochecknoglobals
type AvoidGlobalVariable struct {
	recipe.Base
}

func (r *AvoidGlobalVariable) Name() string {
	return "org.openrewrite.golang.codequality.AvoidGlobalVariable"
}
func (r *AvoidGlobalVariable) DisplayName() string { return "Avoid global variables" }
func (r *AvoidGlobalVariable) Description() string {
	return "Find package-level `var` declarations. Mutable global state makes code harder to test and reason about."
}
func (r *AvoidGlobalVariable) Tags() []string { return []string{"style", "lint"} }

func (r *AvoidGlobalVariable) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{
		{DiagnosticID: "gochecknoglobals", Tool: diagnostic.GolangciLint, HasFix: false},
	}
}

func (r *AvoidGlobalVariable) Editor() recipe.TreeVisitor {
	return visitor.Init(&avoidGlobalVariableVisitor{})
}

type avoidGlobalVariableVisitor struct {
	visitor.GoVisitor
}

func (v *avoidGlobalVariableVisitor) VisitCompilationUnit(cu *tree.CompilationUnit, p any) tree.J {
	cu = v.GoVisitor.VisitCompilationUnit(cu, p).(*tree.CompilationUnit)

	changed := false
	stmts := make([]tree.RightPadded[tree.Statement], len(cu.Statements))
	copy(stmts, cu.Statements)

	for i, stmt := range stmts {
		vd, ok := stmt.Element.(*tree.VariableDeclarations)
		if !ok {
			continue
		}
		if isVarDecl(vd) {
			marked := vd.WithMarkers(tree.MarkupInfo(vd.Markers, "avoid global variable"))
			stmts[i] = tree.RightPadded[tree.Statement]{
				Element: marked,
				After:   stmt.After,
				Markers: stmt.Markers,
			}
			changed = true
		}
	}

	if !changed {
		return cu
	}
	c := *cu
	c.Statements = stmts
	return &c
}

// isVarDecl returns true if the VariableDeclarations has a VarKeyword marker
// and does not have a ConstDecl marker.
func isVarDecl(vd *tree.VariableDeclarations) bool {
	hasVar := tree.FindMarker[tree.VarKeyword](vd.Markers) != nil
	hasConst := tree.FindMarker[tree.ConstDecl](vd.Markers) != nil
	return hasVar && !hasConst
}
