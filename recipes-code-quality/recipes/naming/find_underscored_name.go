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

// FindUnderscoredExportedName finds exported functions using underscores instead
// of camelCase. Go convention: use MixedCaps or mixedCaps, not underscores.
// golangci-lint: revive (var-naming)
type FindUnderscoredExportedName struct {
	recipe.Base
}

func (r *FindUnderscoredExportedName) Name() string {
	return "org.openrewrite.golang.codequality.FindUnderscoredExportedName"
}
func (r *FindUnderscoredExportedName) DisplayName() string {
	return "Find underscored exported names"
}
func (r *FindUnderscoredExportedName) Description() string {
	return "Find exported functions using underscores instead of camelCase. Go convention is to use MixedCaps or mixedCaps."
}
func (r *FindUnderscoredExportedName) Tags() []string { return []string{"naming"} }

func (r *FindUnderscoredExportedName) Editor() recipe.TreeVisitor {
	return visitor.Init(&findUnderscoredExportedNameVisitor{})
}

type findUnderscoredExportedNameVisitor struct {
	visitor.GoVisitor
}

func (v *findUnderscoredExportedNameVisitor) VisitMethodDeclaration(md *tree.MethodDeclaration, p any) tree.J {
	md = v.GoVisitor.VisitMethodDeclaration(md, p).(*tree.MethodDeclaration)

	if md.Name == nil {
		return md
	}

	funcName := md.Name.Name

	// Only check exported names (starts with uppercase).
	firstRune, _ := utf8.DecodeRuneInString(funcName)
	if !unicode.IsUpper(firstRune) {
		return md
	}

	// Check if the name contains an underscore.
	if !strings.Contains(funcName, "_") {
		return md
	}

	md = md.WithName(md.Name.WithMarkers(
		tree.FoundSearchResult(md.Name.Markers, "exported name should use MixedCaps, not underscores"),
	))
	return md
}
