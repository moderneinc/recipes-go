/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"strings"

	"github.com/moderneinc/recipes-go/code-quality/diagnostic"
	"github.com/openrewrite/rewrite/pkg/matcher"
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

var fmtErrorfMatcher = matcher.NewMethodMatcher("fmt Errorf(..)")

// UseErrorsNewForSimpleErrors replaces `fmt.Errorf("static message")` with
// `errors.New("static message")` when the format string contains no format verbs.
// Staticcheck: S1028
type UseErrorsNewForSimpleErrors struct {
	recipe.Base
}

func (r *UseErrorsNewForSimpleErrors) Name() string {
	return "org.openrewrite.golang.codequality.UseErrorsNewForSimpleErrors"
}
func (r *UseErrorsNewForSimpleErrors) DisplayName() string {
	return "Use errors.New for simple errors"
}
func (r *UseErrorsNewForSimpleErrors) Description() string {
	return "Replace `fmt.Errorf(\"static message\")` with `errors.New(\"static message\")` when there are no format verbs."
}
func (r *UseErrorsNewForSimpleErrors) Tags() []string { return []string{"cleanup", "style"} }

func (r *UseErrorsNewForSimpleErrors) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{
		{DiagnosticID: "S1028", Tool: diagnostic.Staticcheck, HasFix: true},
	}
}

func (r *UseErrorsNewForSimpleErrors) Editor() recipe.TreeVisitor {
	return visitor.Init(&useErrorsNewVisitor{})
}

type useErrorsNewVisitor struct {
	visitor.GoVisitor
}

func (v *useErrorsNewVisitor) VisitMethodInvocation(mi *tree.MethodInvocation, p any) tree.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*tree.MethodInvocation)

	// Check: fmt.Errorf(...)
	if !fmtErrorfMatcher.Matches(mi) {
		return mi
	}

	ident, ok := mi.Select.Element.(*tree.Identifier)
	if !ok {
		return mi
	}

	// Must have exactly one argument
	args := mi.Arguments.Elements
	if len(args) != 1 {
		return mi
	}

	// The single argument must be a string literal with no format verbs
	arg := args[0].Element
	lit, ok := arg.(*tree.Literal)
	if !ok || lit.Kind != tree.StringLiteral {
		return mi
	}
	if hasFormatVerb(lit.Source) {
		return mi
	}

	// Build: errors.New(same literal)
	// Reuse the prefix from the original "fmt" identifier.
	prefix := ident.Prefix

	errorsIdent := &tree.Identifier{
		Prefix: prefix,
		Name:   "errors",
	}

	newName := &tree.Identifier{
		Name: "New",
	}

	return &tree.MethodInvocation{
		Select:    &tree.RightPadded[tree.Expression]{Element: errorsIdent, After: mi.Select.After},
		Name:      newName,
		Arguments: mi.Arguments,
	}
}

// hasFormatVerb checks if a Go string literal source (including quotes) contains
// a format verb like %s, %d, %v, %w, etc.
func hasFormatVerb(source string) bool {
	// Strip the surrounding quotes to get the content
	if len(source) < 2 {
		return false
	}
	content := source[1 : len(source)-1]
	return strings.Contains(content, "%")
}
