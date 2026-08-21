/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"unicode"

	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// RemoveEmptyFunction removes unexported free functions with empty bodies and
// no return type. These are dead code -- they do nothing when called. Functions
// with return types are left alone because removing them would break callers.
type RemoveEmptyFunction struct {
	recipe.Base
}

func (r *RemoveEmptyFunction) Name() string {
	return "org.openrewrite.golang.codequality.RemoveEmptyFunction"
}
func (r *RemoveEmptyFunction) DisplayName() string { return "Remove empty functions" }
func (r *RemoveEmptyFunction) Description() string {
	return "Remove unexported free functions with empty bodies and no return type. " +
		"`main` in `package main` and `init` are preserved because the runtime invokes them rather than other code. " +
		"Methods with receivers are preserved because they may implement an interface. " +
		"Exported functions are preserved because removing one breaks importers."
}
func (r *RemoveEmptyFunction) Tags() []string { return []string{"style", "lint"} }

func (r *RemoveEmptyFunction) Editor() recipe.TreeVisitor {
	return visitor.Init(&removeEmptyFunctionVisitor{})
}

type removeEmptyFunctionVisitor struct {
	visitor.GoVisitor
}

func (v *removeEmptyFunctionVisitor) VisitMethodDeclaration(md *java.MethodDeclaration, p any) java.J {
	md = v.GoVisitor.VisitMethodDeclaration(md, p).(*java.MethodDeclaration)

	if md.Name == nil || md.Body == nil {
		return md
	}

	// Only a declaration sitting directly in the file is a free function: a
	// method nests inside the golang.MethodDeclaration carrying its receiver and
	// may implement an interface, and a function literal nests inside the
	// expression whose operand it is.
	cu, isTopLevel := v.Cursor().Parent().Value().(*golang.CompilationUnit)
	if !isTopLevel {
		return md
	}

	// Skip functions with return types -- removing them would break callers.
	if md.ReturnType != nil {
		return md
	}

	name := md.Name.Name
	inPackageMain := cu.PackageDecl != nil && cu.PackageDecl.Element.Name == "main"
	// The runtime invokes `main` and `init` directly; a `package main` that
	// declares no `main` does not link.
	if countRealElements(md.Parameters.Elements) == 0 &&
		(name == "init" || (name == "main" && inPackageMain)) {
		return md
	}

	// Skip exported functions -- removing one is an API break for importers.
	if len(name) > 0 && unicode.IsUpper([]rune(name)[0]) {
		return md
	}

	if countRealElements(md.Body.Statements) > 0 {
		return md
	}

	// Remove the empty function.
	return &java.Empty{}
}

// An empty parameter list or block is represented by a single java.Empty sentinel.
func countRealElements(elements []java.RightPadded[java.Statement]) int {
	count := 0
	for _, e := range elements {
		if _, isEmpty := e.Element.(*java.Empty); !isEmpty {
			count++
		}
	}
	return count
}
