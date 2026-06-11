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

// RemovePackagePrefixFromName finds exported identifiers whose name starts with the
// package name (stuttering). For example, in package `http` a type `HttpClient`
// stutters and should be just `Client`.
// Go convention: the package name should not be repeated in exported identifiers.
// golangci-lint: revive (package-comments)
type RemovePackagePrefixFromName struct {
	recipe.Base
}

func (r *RemovePackagePrefixFromName) Name() string {
	return "org.openrewrite.golang.codequality.RemovePackagePrefixFromName"
}
func (r *RemovePackagePrefixFromName) DisplayName() string {
	return "Remove package prefix from name"
}
func (r *RemovePackagePrefixFromName) Description() string {
	return "Find exported identifiers whose name starts with the package name. Go convention discourages repeating the package name in exported identifiers."
}
func (r *RemovePackagePrefixFromName) Tags() []string { return []string{"naming"} }

func (r *RemovePackagePrefixFromName) Editor() recipe.TreeVisitor {
	return visitor.Init(&removePackagePrefixFromNameVisitor{})
}

type removePackagePrefixFromNameVisitor struct {
	visitor.GoVisitor
	pkgName string
}

func (v *removePackagePrefixFromNameVisitor) VisitCompilationUnit(cu *golang.CompilationUnit, p any) java.J {
	if cu.PackageDecl != nil {
		v.pkgName = cu.PackageDecl.Element.Name
	}
	cu = v.GoVisitor.VisitCompilationUnit(cu, p).(*golang.CompilationUnit)
	return cu
}

func (v *removePackagePrefixFromNameVisitor) VisitMethodDeclaration(md *java.MethodDeclaration, p any) java.J {
	md = v.GoVisitor.VisitMethodDeclaration(md, p).(*java.MethodDeclaration)

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

	// Strip the package prefix from the function name.
	newName := funcName[len(v.pkgName):]
	md = md.WithName(md.Name.WithName(newName))
	return md
}
