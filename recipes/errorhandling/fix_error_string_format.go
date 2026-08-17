/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling

import (
	"strconv"
	"unicode"

	"github.com/moderneinc/recipes-go/diagnostic"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// FixErrorStringFormat lowercases the leading word of an `errors.New` or
// `fmt.Errorf` message and drops its trailing punctuation, the two ST1005
// violations AuditErrorStringFormat reports.
type FixErrorStringFormat struct {
	recipe.Base
}

func (r *FixErrorStringFormat) Name() string {
	return "org.openrewrite.golang.codequality.FixErrorStringFormat"
}
func (r *FixErrorStringFormat) DisplayName() string { return "Fix error string format" }
func (r *FixErrorStringFormat) Description() string {
	return "Lowercase the leading word of `errors.New` and `fmt.Errorf` messages and remove trailing punctuation, so the message reads correctly when a caller wraps it in a larger one (staticcheck ST1005)."
}
func (r *FixErrorStringFormat) Tags() []string { return []string{"error-handling", "lint"} }

func (r *FixErrorStringFormat) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{
		{DiagnosticID: "ST1005", Tool: diagnostic.Staticcheck, HasFix: false},
	}
}

func (r *FixErrorStringFormat) Editor() recipe.TreeVisitor {
	return visitor.Init(&fixErrorStringFormatVisitor{})
}

type fixErrorStringFormatVisitor struct {
	visitor.GoVisitor
}

func (v *fixErrorStringFormatVisitor) VisitMethodInvocation(mi *java.MethodInvocation, p any) java.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*java.MethodInvocation)

	if errorConstructorName(mi) == "" {
		return mi
	}
	args := mi.Arguments.Elements
	if len(args) == 0 {
		return mi
	}
	lit, ok := args[0].Element.(*java.Literal)
	if !ok {
		return mi
	}
	msg, err := strconv.Unquote(lit.Source)
	if err != nil || msg == "" {
		return mi
	}

	fixed := normalizeErrorString(msg)
	if fixed == msg {
		return mi
	}

	// Quoting the fixed message reproduces the escapes the source had, since the
	// message came from unquoting it.
	newLit := *lit
	newLit.Value = fixed
	newLit.Source = strconv.Quote(fixed)

	newArgs := make([]java.RightPadded[java.Expression], len(args))
	copy(newArgs, args)
	newArgs[0] = java.RightPadded[java.Expression]{
		Element: &newLit, After: args[0].After, Markers: args[0].Markers,
	}

	newMi := *mi
	newMi.Arguments = java.Container[java.Expression]{
		Before:   mi.Arguments.Before,
		Elements: newArgs,
		Markers:  mi.Arguments.Markers,
	}
	return &newMi
}

func normalizeErrorString(msg string) string {
	runes := []rune(msg)
	if startsWithOrdinaryCapital(runes) {
		runes[0] = unicode.ToLower(runes[0])
	}
	if endsWithPunctuation(runes) {
		runes = runes[:len(runes)-1]
	}
	return string(runes)
}
