/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification

import (
	"github.com/moderneinc/recipes-go/recipes-code-quality/diagnostic"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/matcher"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

var timeNowMatcher = matcher.NewMethodMatcher("time Now()")

// ReplaceTimeSinceWithSince replaces `time.Now().Sub(t)` with `time.Since(t)`.
// Staticcheck: S1012
type ReplaceTimeSinceWithSince struct {
	recipe.Base
}

func (r *ReplaceTimeSinceWithSince) Name() string {
	return "org.openrewrite.golang.codequality.ReplaceTimeSinceWithSince"
}
func (r *ReplaceTimeSinceWithSince) DisplayName() string { return "Replace time.Now().Sub(t) with time.Since(t)" }
func (r *ReplaceTimeSinceWithSince) Description() string {
	return "Replace `time.Now().Sub(t)` with `time.Since(t)` for clarity."
}
func (r *ReplaceTimeSinceWithSince) Tags() []string { return []string{"cleanup", "simplification"} }

func (r *ReplaceTimeSinceWithSince) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{
		{DiagnosticID: "S1012", Tool: diagnostic.Staticcheck, HasFix: true},
	}
}

func (r *ReplaceTimeSinceWithSince) Editor() recipe.TreeVisitor {
	return visitor.Init(&replaceTimeSinceVisitor{})
}

type replaceTimeSinceVisitor struct {
	visitor.GoVisitor
}

func (v *replaceTimeSinceVisitor) VisitMethodInvocation(mi *tree.MethodInvocation, p any) tree.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*tree.MethodInvocation)

	// Match: <something>.Sub(arg)
	if mi.Name.Name != "Sub" || mi.Select == nil {
		return mi
	}

	// The select must be time.Now() — a method invocation with no args
	selectExpr := mi.Select.Element
	nowCall, ok := selectExpr.(*tree.MethodInvocation)
	if !ok || !timeNowMatcher.Matches(nowCall) {
		return mi
	}

	timeIdent, ok := nowCall.Select.Element.(*tree.Identifier)
	if !ok {
		return mi
	}

	// Sub must have exactly one argument
	args := mi.Arguments.Elements
	subArg := getOnlyArg(args)
	if subArg == nil {
		return mi
	}

	// Build: time.Since(arg)
	// Reuse the existing "time" identifier and its prefix for leading whitespace.
	prefix := timeIdent.Prefix

	newTimeIdent := &tree.Identifier{
		Prefix: prefix,
		Name:   "time",
	}

	sinceIdent := &tree.Identifier{
		Name: "Since",
	}

	return &tree.MethodInvocation{
		Select:    &tree.RightPadded[tree.Expression]{Element: newTimeIdent, After: mi.Select.After},
		Name:      sinceIdent,
		Arguments: mi.Arguments, // reuse the argument list (contains the original arg)
	}
}

// getOnlyArg returns the single real argument from the argument list,
// skipping any Empty sentinel. Returns nil if there isn't exactly one arg.
func getOnlyArg(args []tree.RightPadded[tree.Expression]) tree.Expression {
	var real []tree.Expression
	for _, a := range args {
		if _, isEmpty := a.Element.(*tree.Empty); !isEmpty {
			real = append(real, a.Element)
		}
	}
	if len(real) != 1 {
		return nil
	}
	return real[0]
}
