/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/moderneinc/recipes-go/diagnostic"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// AddExportedFuncComment adds a stub doc comment `// FuncName ...` to exported
// functions and methods that are missing one.
// golangci-lint: revive (exported)
type AddExportedFuncComment struct {
	recipe.Base
}

func (r *AddExportedFuncComment) Name() string {
	return "org.openrewrite.golang.codequality.AddExportedFuncComment"
}
func (r *AddExportedFuncComment) DisplayName() string {
	return "Add exported func comment"
}
func (r *AddExportedFuncComment) Description() string {
	return "Add a stub doc comment to exported functions and methods that lack one."
}
func (r *AddExportedFuncComment) Tags() []string {
	return []string{"style", "lint"}
}

func (r *AddExportedFuncComment) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{
		{DiagnosticID: "ST1020", Tool: diagnostic.Staticcheck, HasFix: true},
		{DiagnosticID: "exported", Tool: diagnostic.GolangciLint, HasFix: true},
	}
}

func (r *AddExportedFuncComment) Editor() recipe.TreeVisitor {
	return visitor.Init(&addExportedFuncCommentVisitor{})
}

type addExportedFuncCommentVisitor struct {
	visitor.GoVisitor
}

func (v *addExportedFuncCommentVisitor) VisitMethodDeclaration(md *java.MethodDeclaration, p any) java.J {
	md = v.GoVisitor.VisitMethodDeclaration(md, p).(*java.MethodDeclaration)

	// Methods (with a receiver) are wrapped in golang.MethodDeclaration, which
	// owns the prefix; they are handled by VisitGoMethodDeclaration. Here we only
	// handle free functions, whose prefix lives on the declaration itself.
	if _, wrapped := v.Cursor().Parent().Value().(*golang.MethodDeclaration); wrapped {
		return md
	}
	if md.Name == nil {
		return md
	}

	newPrefix, ok := addDocComment(md.Name.Name, md.Prefix)
	if !ok {
		return md
	}
	return md.WithPrefix(newPrefix)
}

func (v *addExportedFuncCommentVisitor) VisitGoMethodDeclaration(md *golang.MethodDeclaration, p any) java.J {
	md = v.GoVisitor.VisitGoMethodDeclaration(md, p).(*golang.MethodDeclaration)

	if md.Declaration == nil || md.Declaration.Name == nil {
		return md
	}

	// The wrapper owns the prefix (and thus any doc comment) before `func`.
	newPrefix, ok := addDocComment(md.Declaration.Name.Name, md.Prefix)
	if !ok {
		return md
	}
	return md.WithPrefix(newPrefix)
}

// addDocComment returns prefix with a stub `// Name ...` doc comment appended and
// true, when name is exported and prefix does not already carry a matching doc
// comment. Otherwise it returns prefix unchanged and false.
func addDocComment(name string, prefix java.Space) (java.Space, bool) {
	// Only exported names (starting with an uppercase letter) get a doc comment.
	firstRune, _ := utf8.DecodeRuneInString(name)
	if !unicode.IsUpper(firstRune) {
		return prefix, false
	}

	if doc, ok := docBlock(prefix.Comments); ok && strings.HasPrefix(doc.Text, "// "+name) {
		return prefix, false
	}

	// Add a stub doc comment: // Name ...
	// The Space model: Whitespace is emitted first, then each Comment (Text + Suffix).
	// After the last comment's suffix, the node keyword (`func`) follows.
	// We need the comment on its own line, indented the same as the func keyword.
	// The comment suffix is "\n" + indent so the func keyword starts at the correct column.
	commentText := "// " + name + " ..."
	indent := prefix.Indent()
	comment := java.Comment{Kind: java.LineComment, Text: commentText, Suffix: "\n" + indent}
	return java.Space{
		Whitespace: prefix.Whitespace,
		Comments:   append(prefix.Comments, comment),
	}, true
}

// docBlock returns the first comment of the run directly above the declaration
// with no blank line inside it or before the declaration. Go attaches
// documentation only to such a run, so a comment above a blank line documents
// nothing.
func docBlock(comments []java.Comment) (java.Comment, bool) {
	if len(comments) == 0 || separatedByBlankLine(comments[len(comments)-1].Suffix) {
		return java.Comment{}, false
	}
	first := len(comments) - 1
	for first > 0 && !separatedByBlankLine(comments[first-1].Suffix) {
		first--
	}
	return comments[first], true
}

func separatedByBlankLine(suffix string) bool {
	return strings.Count(suffix, "\n") > 1
}
