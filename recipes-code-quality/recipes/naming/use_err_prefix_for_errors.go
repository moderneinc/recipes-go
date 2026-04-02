/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package naming

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// UseErrPrefixForErrors finds package-level error variables that do not follow
// the `ErrFoo` naming convention. Go convention: sentinel errors should be
// named with an "Err" prefix.
type UseErrPrefixForErrors struct {
	recipe.Base
}

func (r *UseErrPrefixForErrors) Name() string {
	return "org.openrewrite.golang.codequality.UseErrPrefixForErrors"
}
func (r *UseErrPrefixForErrors) DisplayName() string { return "Use Err prefix for errors" }
func (r *UseErrPrefixForErrors) Description() string {
	return "Find package-level error variables not following the `ErrFoo` naming convention. Go convention is to prefix sentinel errors with \"Err\"."
}
func (r *UseErrPrefixForErrors) Tags() []string { return []string{"naming"} }

func (r *UseErrPrefixForErrors) Editor() recipe.TreeVisitor {
	return visitor.Init(&useErrPrefixForErrorsVisitor{})
}

type useErrPrefixForErrorsVisitor struct {
	visitor.GoVisitor
}

func (v *useErrPrefixForErrorsVisitor) VisitCompilationUnit(cu *tree.CompilationUnit, p any) tree.J {
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
func (v *useErrPrefixForErrorsVisitor) checkVarDecl(vd *tree.VariableDeclarations) *tree.VariableDeclarations {
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

		// Rename by prepending "Err" and capitalizing the first letter.
		newName := toErrName(decl.Name.Name)
		renamed := decl.WithName(decl.Name.WithName(newName))
		vars[j] = tree.RightPadded[*tree.VariableDeclarator]{
			Element: renamed,
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

// isErrorConstructor checks if an expression is a call to errors.New(...) or
// fmt.Errorf(...).
func isErrorConstructor(expr tree.Expression) bool {
	mi, ok := expr.(*tree.MethodInvocation)
	if !ok {
		return false
	}
	if mi.Select == nil {
		return false
	}
	sel, ok := mi.Select.Element.(*tree.Identifier)
	if !ok {
		return false
	}
	return (sel.Name == "errors" && mi.Name.Name == "New") ||
		(sel.Name == "fmt" && mi.Name.Name == "Errorf")
}

// toErrName converts a variable name to an "Err"-prefixed name.
// If the name starts with lowercase, capitalize and prepend "Err".
// If already capitalized, just prepend "Err".
func toErrName(name string) string {
	r, size := utf8.DecodeRuneInString(name)
	if unicode.IsLower(r) {
		return "Err" + string(unicode.ToUpper(r)) + name[size:]
	}
	return "Err" + name
}
