/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"strconv"
	"strings"

	"github.com/openrewrite/rewrite/rewrite-go/pkg/matcher"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

var (
	regexpCompileMatcher     = matcher.NewMethodMatcher("regexp Compile(..)")
	regexpMustCompileMatcher = matcher.NewMethodMatcher("regexp MustCompile(..)")
)

// PreferRawStringForRegex replaces interpreted string literals containing
// backslash escapes with raw string literals in calls to `regexp.Compile` or
// `regexp.MustCompile`. Raw string literals (backtick-quoted) are preferred
// for regex patterns as they avoid double-escaping.
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
	return "Replace interpreted string literals containing backslash escapes with raw string literals in `regexp.Compile` or `regexp.MustCompile` calls."
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

	// Unquote the interpreted string to get the actual regex value,
	// then wrap it in backticks for a raw string literal.
	unquoted, err := strconv.Unquote(lit.Source)
	if err != nil {
		return mi
	}

	// Raw strings cannot contain backticks; bail out if the value has one.
	if strings.Contains(unquoted, "`") {
		return mi
	}

	newSource := "`" + unquoted + "`"
	newLit := *lit
	newLit.Source = newSource
	newLit.Value = unquoted

	newArgs := make([]tree.RightPadded[tree.Expression], len(args))
	copy(newArgs, args)
	newArgs[0] = tree.RightPadded[tree.Expression]{Element: &newLit, After: args[0].After, Markers: args[0].Markers}

	newMi := *mi
	newMi.Arguments.Elements = newArgs
	return &newMi
}
