/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/moderneinc/recipes-go/code-quality/diagnostic"
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// ExportedFuncShouldHaveComment finds exported functions and methods that
// are missing a doc comment starting with the function name.
// golangci-lint: revive (exported)
type ExportedFuncShouldHaveComment struct {
	recipe.Base
}

func (r *ExportedFuncShouldHaveComment) Name() string {
	return "org.openrewrite.golang.codequality.ExportedFuncShouldHaveComment"
}
func (r *ExportedFuncShouldHaveComment) DisplayName() string {
	return "Exported function should have comment"
}
func (r *ExportedFuncShouldHaveComment) Description() string {
	return "Find exported functions and methods that lack a doc comment starting with the function name."
}
func (r *ExportedFuncShouldHaveComment) Tags() []string {
	return []string{"style", "lint"}
}

func (r *ExportedFuncShouldHaveComment) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{
		{DiagnosticID: "exported", Tool: diagnostic.GolangciLint, HasFix: false},
	}
}

func (r *ExportedFuncShouldHaveComment) Editor() recipe.TreeVisitor {
	return visitor.Init(&exportedFuncShouldHaveCommentVisitor{})
}

type exportedFuncShouldHaveCommentVisitor struct {
	visitor.GoVisitor
}

func (v *exportedFuncShouldHaveCommentVisitor) VisitMethodDeclaration(md *tree.MethodDeclaration, p any) tree.J {
	md = v.GoVisitor.VisitMethodDeclaration(md, p).(*tree.MethodDeclaration)

	if md.Name == nil {
		return md
	}

	funcName := md.Name.Name

	// Check if the function name starts with an uppercase letter (exported).
	firstRune, _ := utf8.DecodeRuneInString(funcName)
	if !unicode.IsUpper(firstRune) {
		return md
	}

	// Check if there is a doc comment in the prefix space of the method declaration.
	// A proper doc comment is the last comment before the `func` keyword and
	// starts with "// FuncName".
	comments := md.Prefix.Comments
	if len(comments) > 0 {
		lastComment := comments[len(comments)-1]
		expectedPrefix := "// " + funcName
		if strings.HasPrefix(lastComment.Text, expectedPrefix) {
			return md
		}
	}

	// Mark the function name identifier with a search result.
	md = md.WithName(md.Name.WithMarkers(
		tree.FoundSearchResult(md.Name.Markers, "exported function "+funcName+" should have a comment"),
	))
	return md
}
