/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package naming

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindStutteringName finds exported identifiers whose name starts with the
// package name (stuttering). For example, in package `http` a type `HttpClient`
// stutters and should be just `Client`.
// Go convention: the package name should not be repeated in exported identifiers.
// golangci-lint: revive (package-comments)
type FindStutteringName struct {
	recipe.Base
}

func (r *FindStutteringName) Name() string {
	return "org.openrewrite.golang.codequality.FindStutteringName"
}
func (r *FindStutteringName) DisplayName() string { return "Find stuttering names" }
func (r *FindStutteringName) Description() string {
	return "Find exported identifiers whose name starts with the package name. Go convention discourages repeating the package name in exported identifiers."
}
func (r *FindStutteringName) Tags() []string { return []string{"naming"} }

func (r *FindStutteringName) Editor() recipe.TreeVisitor {
	return visitor.Init(&findStutteringNameVisitor{})
}

type findStutteringNameVisitor struct {
	visitor.GoVisitor
	pkgName string
}

func (v *findStutteringNameVisitor) VisitCompilationUnit(cu *tree.CompilationUnit, p any) tree.J {
	if cu.PackageDecl != nil {
		v.pkgName = cu.PackageDecl.Element.Name
	}
	cu = v.GoVisitor.VisitCompilationUnit(cu, p).(*tree.CompilationUnit)
	return cu
}

func (v *findStutteringNameVisitor) VisitMethodDeclaration(md *tree.MethodDeclaration, p any) tree.J {
	md = v.GoVisitor.VisitMethodDeclaration(md, p).(*tree.MethodDeclaration)

	if md.Name == nil || v.pkgName == "" {
		return md
	}

	funcName := md.Name.Name

	// Only check exported names (starts with uppercase).
	firstRune, _ := utf8.DecodeRuneInString(funcName)
	if !unicode.IsUpper(firstRune) {
		return md
	}

	// Check if the function name starts with the package name (case-insensitive).
	if len(funcName) <= len(v.pkgName) {
		return md
	}
	if !strings.EqualFold(funcName[:len(v.pkgName)], v.pkgName) {
		return md
	}

	md = md.WithName(md.Name.WithMarkers(
		tree.FoundSearchResult(md.Name.Markers, "name stutters with package name"),
	))
	return md
}
