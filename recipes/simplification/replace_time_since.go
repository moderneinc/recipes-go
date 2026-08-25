/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification

import (
	"fmt"

	"github.com/moderneinc/recipes-go/diagnostic"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/matcher"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/template"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
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
func (r *ReplaceTimeSinceWithSince) DisplayName() string {
	return "Replace time.Now().Sub(t) with time.Since(t)"
}
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

func (v *replaceTimeSinceVisitor) VisitMethodInvocation(mi *java.MethodInvocation, p any) java.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*java.MethodInvocation)

	// Match: <something>.Sub(arg)
	if mi.Name.Name != "Sub" || mi.Select == nil {
		return mi
	}

	// The select must be time.Now() — a method invocation with no args
	selectExpr := mi.Select.Element
	nowCall, ok := selectExpr.(*java.MethodInvocation)
	if !ok || !timeNowMatcher.Matches(nowCall) {
		return mi
	}

	if _, ok := nowCall.Select.Element.(*java.Identifier); !ok {
		return mi
	}

	// Sub must have exactly one argument
	args := mi.Arguments.Elements
	subArg := getOnlyArg(args)
	if subArg == nil {
		return mi
	}

	replaced := timeSinceTmpl.Apply(v.Cursor(), template.NewMatchResult().Bind(timeSinceArg, subArg))
	if replaced == nil {
		return mi
	}
	return replaced
}

var (
	timeSinceArg  = template.Expr("t").WithType("time.Time")
	timeSinceTmpl = template.ExpressionTemplate(fmt.Sprintf("time.Since(%s)", timeSinceArg)).
			Captures(timeSinceArg).
			Imports("time").
			Build()
)

// getOnlyArg returns the single real argument from the argument list,
// skipping any Empty sentinel. Returns nil if there isn't exactly one arg.
func getOnlyArg(args []java.RightPadded[java.Expression]) java.Expression {
	var real []java.Expression
	for _, a := range args {
		if _, isEmpty := a.Element.(*java.Empty); !isEmpty {
			real = append(real, a.Element)
		}
	}
	if len(real) != 1 {
		return nil
	}
	return real[0]
}
