/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package naming

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
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

func (v *useErrPrefixForErrorsVisitor) VisitCompilationUnit(cu *golang.CompilationUnit, p any) java.J {
	cu = v.GoVisitor.VisitCompilationUnit(cu, p).(*golang.CompilationUnit)

	changed := false
	stmts := make([]java.RightPadded[java.Statement], len(cu.Statements))
	copy(stmts, cu.Statements)

	for i, stmt := range stmts {
		vd, ok := stmt.Element.(*java.VariableDeclarations)
		if !ok {
			continue
		}

		// Only check `var` declarations (not `const`).
		hasVar := java.FindMarker[golang.VarKeyword](vd.Markers) != nil
		hasConst := java.FindMarker[golang.ConstDecl](vd.Markers) != nil
		if !hasVar || hasConst {
			continue
		}

		markedVD := v.checkVarDecl(vd)
		if markedVD != vd {
			stmts[i] = java.RightPadded[java.Statement]{
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
func (v *useErrPrefixForErrorsVisitor) checkVarDecl(vd *java.VariableDeclarations) *java.VariableDeclarations {
	changed := false
	vars := make([]java.RightPadded[*java.VariableDeclarator], len(vd.Variables))
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
		vars[j] = java.RightPadded[*java.VariableDeclarator]{
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
func isErrorConstructor(expr java.Expression) bool {
	mi, ok := expr.(*java.MethodInvocation)
	if !ok {
		return false
	}
	if mi.Select == nil {
		return false
	}
	sel, ok := mi.Select.Element.(*java.Identifier)
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
