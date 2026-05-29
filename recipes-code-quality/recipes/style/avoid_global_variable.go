/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/moderneinc/recipes-go/recipes-code-quality/diagnostic"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
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

func (v *avoidGlobalVariableVisitor) VisitCompilationUnit(cu *golang.CompilationUnit, p any) java.J {
	cu = v.GoVisitor.VisitCompilationUnit(cu, p).(*golang.CompilationUnit)

	changed := false
	stmts := make([]java.RightPadded[java.Statement], len(cu.Statements))
	copy(stmts, cu.Statements)

	for i, stmt := range stmts {
		vd, ok := stmt.Element.(*java.VariableDeclarations)
		if !ok {
			continue
		}
		if isVarDecl(vd) {
			marked := vd.WithMarkers(java.MarkupInfo(vd.Markers, "avoid global variable"))
			stmts[i] = java.RightPadded[java.Statement]{
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
func isVarDecl(vd *java.VariableDeclarations) bool {
	hasVar := java.FindMarker[golang.VarKeyword](vd.Markers) != nil
	hasConst := java.FindMarker[golang.ConstDecl](vd.Markers) != nil
	return hasVar && !hasConst
}
