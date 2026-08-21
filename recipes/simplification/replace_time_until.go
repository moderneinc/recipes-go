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

var untilTimeNowMatcher = matcher.NewMethodMatcher("time Now()")

// ReplaceTimeUntilWithUntil replaces `t.Sub(time.Now())` with `time.Until(t)`.
// Staticcheck: S1024
type ReplaceTimeUntilWithUntil struct {
	recipe.Base
}

func (r *ReplaceTimeUntilWithUntil) Name() string {
	return "org.openrewrite.golang.codequality.ReplaceTimeUntilWithUntil"
}
func (r *ReplaceTimeUntilWithUntil) DisplayName() string {
	return "Replace t.Sub(time.Now()) with time.Until(t)"
}
func (r *ReplaceTimeUntilWithUntil) Description() string {
	return "Replace `t.Sub(time.Now())` with `time.Until(t)` for clarity."
}
func (r *ReplaceTimeUntilWithUntil) Tags() []string { return []string{"cleanup", "simplification"} }

func (r *ReplaceTimeUntilWithUntil) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{
		{DiagnosticID: "S1024", Tool: diagnostic.Staticcheck, HasFix: true},
	}
}

func (r *ReplaceTimeUntilWithUntil) Editor() recipe.TreeVisitor {
	return visitor.Init(&replaceTimeUntilVisitor{})
}

type replaceTimeUntilVisitor struct {
	visitor.GoVisitor
}

func (v *replaceTimeUntilVisitor) VisitMethodInvocation(mi *java.MethodInvocation, p any) java.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*java.MethodInvocation)

	// Match: <receiver>.Sub(arg)
	if mi.Name.Name != "Sub" || mi.Select == nil {
		return mi
	}

	// The argument must be time.Now()
	args := mi.Arguments.Elements
	var argExpr java.Expression
	for _, a := range args {
		if _, isEmpty := a.Element.(*java.Empty); !isEmpty {
			if argExpr != nil {
				return mi // more than one real arg
			}
			argExpr = a.Element
		}
	}
	if argExpr == nil {
		return mi
	}

	nowCall, ok := argExpr.(*java.MethodInvocation)
	if !ok || !untilTimeNowMatcher.Matches(nowCall) {
		return mi
	}

	// The receiver (mi.Select) is the time value `t` to pass to time.Until(t).
	replaced := timeUntilTmpl.Apply(v.Cursor(), template.NewMatchResult().Bind(timeUntilArg, mi.Select.Element))
	if replaced == nil {
		return mi
	}
	return replaced
}

var (
	timeUntilArg  = template.Expr("t").WithType("time.Time")
	timeUntilTmpl = template.ExpressionTemplate(fmt.Sprintf("time.Until(%s)", timeUntilArg)).
			Captures(timeUntilArg).
			Imports("time").
			Build()
)
