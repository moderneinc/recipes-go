/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package naming

import (
	"strings"

	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindMisnamedErrorVar finds package-level error variables that do not follow
// the `ErrFoo` naming convention. Go convention: sentinel errors should be
// named with an "Err" prefix.
type FindMisnamedErrorVar struct {
	recipe.Base
}

func (r *FindMisnamedErrorVar) Name() string {
	return "org.openrewrite.golang.codequality.FindMisnamedErrorVar"
}
func (r *FindMisnamedErrorVar) DisplayName() string { return "Find misnamed error variables" }
func (r *FindMisnamedErrorVar) Description() string {
	return "Find package-level error variables not following the `ErrFoo` naming convention. Go convention is to prefix sentinel errors with \"Err\"."
}
func (r *FindMisnamedErrorVar) Tags() []string { return []string{"naming"} }

func (r *FindMisnamedErrorVar) Editor() recipe.TreeVisitor {
	return visitor.Init(&findMisnamedErrorVarVisitor{})
}

type findMisnamedErrorVarVisitor struct {
	visitor.GoVisitor
}

func (v *findMisnamedErrorVarVisitor) VisitCompilationUnit(cu *tree.CompilationUnit, p any) tree.J {
	cu = v.GoVisitor.VisitCompilationUnit(cu, p).(*tree.CompilationUnit)

	changed := false
	stmts := make([]tree.RightPadded[tree.Statement], len(cu.Statements))
	copy(stmts, cu.Statements)

	for i, stmt := range stmts {
		vd, ok := stmt.Element.(*tree.VariableDeclarations)
		if !ok {
			continue
		}

		// Only check `var` declarations (not `const`).
		hasVar := tree.FindMarker[tree.VarKeyword](vd.Markers) != nil
		hasConst := tree.FindMarker[tree.ConstDecl](vd.Markers) != nil
		if !hasVar || hasConst {
			continue
		}

		markedVD := v.checkVarDecl(vd)
		if markedVD != vd {
			stmts[i] = tree.RightPadded[tree.Statement]{
				Element: markedVD,
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

// checkVarDecl checks a top-level var declaration for misnamed error variables.
func (v *findMisnamedErrorVarVisitor) checkVarDecl(vd *tree.VariableDeclarations) *tree.VariableDeclarations {
	changed := false
	vars := make([]tree.RightPadded[*tree.VariableDeclarator], len(vd.Variables))
	copy(vars, vd.Variables)

	for j, varRP := range vars {
		decl := varRP.Element
		if decl.Name == nil || decl.Initializer == nil {
			continue
		}

		// Check if the initializer is errors.New(...) or fmt.Errorf(...).
		if !isErrorConstructor(decl.Initializer.Element) {
			continue
		}

		// Check if the var name starts with "Err".
		if strings.HasPrefix(decl.Name.Name, "Err") {
			continue
		}

		marked := decl.WithName(decl.Name.WithMarkers(
			tree.FoundSearchResult(decl.Name.Markers, "error variable should start with Err"),
		))
		vars[j] = tree.RightPadded[*tree.VariableDeclarator]{
			Element: marked,
			After:   varRP.After,
			Markers: varRP.Markers,
		}
		changed = true
	}

	if !changed {
		return vd
	}
	c := *vd
	c.Variables = vars
	return &c
}

// isErrorConstructor returns true if the expression is a call to errors.New or fmt.Errorf.
func isErrorConstructor(expr tree.Expression) bool {
	mi, ok := expr.(*tree.MethodInvocation)
	if !ok || mi.Select == nil {
		return false
	}

	sel, ok := mi.Select.Element.(*tree.Identifier)
	if !ok {
		return false
	}

	if sel.Name == "errors" && mi.Name.Name == "New" {
		return true
	}
	if sel.Name == "fmt" && mi.Name.Name == "Errorf" {
		return true
	}

	return false
}
