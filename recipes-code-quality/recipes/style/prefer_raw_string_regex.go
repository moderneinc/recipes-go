/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"strings"

	"github.com/openrewrite/rewrite/pkg/matcher"
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

var (
	regexpCompileMatcher     = matcher.NewMethodMatcher("regexp Compile(..)")
	regexpMustCompileMatcher = matcher.NewMethodMatcher("regexp MustCompile(..)")
)

// PreferRawStringForRegex finds calls to `regexp.Compile` or
// `regexp.MustCompile` where the pattern argument is a interpreted string
// literal containing backslash escapes. Raw string literals (backtick-quoted)
// are preferred for regex patterns as they avoid double-escaping.
type PreferRawStringForRegex struct {
	recipe.Base
}

func (r *PreferRawStringForRegex) Name() string {
	return "org.openrewrite.golang.codequality.PreferRawStringForRegex"
}
func (r *PreferRawStringForRegex) DisplayName() string {
	return "Prefer raw string literals for regex patterns"
}
func (r *PreferRawStringForRegex) Description() string {
	return "Find `regexp.Compile` or `regexp.MustCompile` calls where the pattern uses an interpreted string with backslash escapes instead of a raw string literal."
}
func (r *PreferRawStringForRegex) Tags() []string { return []string{"style"} }

func (r *PreferRawStringForRegex) Editor() recipe.TreeVisitor {
	return visitor.Init(&preferRawStringRegexVisitor{})
}

type preferRawStringRegexVisitor struct {
	visitor.GoVisitor
}

func (v *preferRawStringRegexVisitor) VisitMethodInvocation(mi *tree.MethodInvocation, p any) tree.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*tree.MethodInvocation)

	if !regexpCompileMatcher.Matches(mi) && !regexpMustCompileMatcher.Matches(mi) {
		return mi
	}

	// Get the first argument.
	args := mi.Arguments.Elements
	if len(args) == 0 {
		return mi
	}

	firstArg := args[0].Element
	lit, ok := firstArg.(*tree.Literal)
	if !ok || lit.Kind != tree.StringLiteral {
		return mi
	}

	// Check that it is an interpreted string (starts with `"`, not a backtick).
	if !strings.HasPrefix(lit.Source, "\"") {
		return mi
	}

	// Check if it contains a backslash escape.
	content := lit.Source[1 : len(lit.Source)-1]
	if !strings.Contains(content, "\\") {
		return mi
	}

	mi = mi.WithMarkers(tree.FoundSearchResult(mi.Markers, "prefer raw string literal for regex pattern"))
	return mi
}
