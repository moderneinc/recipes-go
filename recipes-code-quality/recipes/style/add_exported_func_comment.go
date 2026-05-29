/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
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

func (r *AddExportedFuncComment) Editor() recipe.TreeVisitor {
	return visitor.Init(&addExportedFuncCommentVisitor{})
}

type addExportedFuncCommentVisitor struct {
	visitor.GoVisitor
}

func (v *addExportedFuncCommentVisitor) VisitMethodDeclaration(md *java.MethodDeclaration, p any) java.J {
	md = v.GoVisitor.VisitMethodDeclaration(md, p).(*java.MethodDeclaration)

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

	// Add a stub doc comment: // FuncName ...
	// The Space model: Whitespace is emitted first, then each Comment (Text + Suffix).
	// After the last comment's suffix, the node keyword (`func`) follows.
	// We need the comment on its own line, indented the same as the func keyword.
	// The comment suffix is "\n" + indent so the func keyword starts at the correct column.
	commentText := "// " + funcName + " ..."
	indent := md.Prefix.Indent()
	comment := java.Comment{Kind: java.LineComment, Text: commentText, Suffix: "\n" + indent}

	newComments := append(md.Prefix.Comments, comment)
	md = md.WithPrefix(java.Space{
		Whitespace: md.Prefix.Whitespace,
		Comments:   newComments,
	})
	return md
}
