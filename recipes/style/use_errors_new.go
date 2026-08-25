/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"fmt"
	"strings"

	"github.com/moderneinc/recipes-go/diagnostic"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/matcher"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	recipegolang "github.com/openrewrite/rewrite/rewrite-go/pkg/recipe/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/template"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
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

func (v *useErrorsNewVisitor) VisitMethodInvocation(mi *java.MethodInvocation, p any) java.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*java.MethodInvocation)

	// Check: fmt.Errorf(...)
	if !fmtErrorfMatcher.Matches(mi) {
		return mi
	}

	if _, ok := mi.Select.Element.(*java.Identifier); !ok {
		return mi
	}

	// Must have exactly one argument
	args := mi.Arguments.Elements
	if len(args) != 1 {
		return mi
	}

	// The single argument must be a string literal with no format verbs
	arg := args[0].Element
	lit, ok := arg.(*java.Literal)
	if !ok || !matcher.IsString(lit.Type) {
		return mi
	}
	if hasFormatVerb(lit.Source) {
		return mi
	}

	// Removing `fmt` ahead of adding `errors` keeps the one-line `import "errors"`
	// for a file whose only import was `fmt`; the other order yields a block.
	recipegolang.MaybeRemoveImport(v, "fmt")
	recipegolang.MaybeAddImport(v, "errors", nil, false)

	replaced := errorsNewTmpl.Apply(v.Cursor(), template.NewMatchResult().Bind(errorsNewMsg, lit))
	if replaced == nil {
		return mi
	}
	return replaced
}

var (
	errorsNewMsg  = template.Expr("msg").WithType("string")
	errorsNewTmpl = template.ExpressionTemplate(fmt.Sprintf("errors.New(%s)", errorsNewMsg)).
			Captures(errorsNewMsg).
			Imports("errors").
			Build()
)

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
