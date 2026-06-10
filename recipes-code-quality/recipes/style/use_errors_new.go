/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"strings"

	"github.com/moderneinc/recipes-go/recipes-code-quality/diagnostic"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/matcher"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	recipegolang "github.com/openrewrite/rewrite/rewrite-go/pkg/recipe/golang"
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
	changed bool
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

	// The rewrite strands the now-unused `fmt` import and introduces a reference
	// to `errors`. Queue the unused-import cleanup BEFORE adding `errors`:
	// RemoveUnusedImports derives referenced packages from type-attributed
	// identifiers, so it cannot see the freshly built (untyped) `errors`
	// reference and would drop the import if it ran afterwards. Queued first it
	// runs first — removing only the stranded `fmt` — then the unconditional
	// AddImport adds `errors`.
	if !v.changed {
		v.DoAfterVisit((&recipegolang.RemoveUnusedImports{}).Editor())
		v.changed = true
	}
	recipegolang.MaybeAddImport(v, "errors", nil, false)

	// Build: errors.New(same literal)
	// The leading whitespace lives on the outermost element (the invocation),
	// so carry the original invocation's prefix onto the replacement.
	errorsIdent := &java.Identifier{
		Name: "errors",
	}

	newName := &java.Identifier{
		Name: "New",
	}

	return &java.MethodInvocation{
		Prefix:    mi.Prefix,
		Select:    &java.RightPadded[java.Expression]{Element: errorsIdent, After: mi.Select.After},
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
