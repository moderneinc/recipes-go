/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package performance

import (
	"fmt"
	"github.com/moderneinc/recipes-go/diagnostic"

	"github.com/openrewrite/rewrite/rewrite-go/pkg/matcher"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	recipegolang "github.com/openrewrite/rewrite/rewrite-go/pkg/recipe/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/template"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

var fbB = template.Expr("fbB")

var (
	formatBoolPattern = template.Expression(fmt.Sprintf(`fmt.Sprintf("%%t", %s)`, fbB)).
				Captures(fbB).Imports("fmt").Build()
	formatBoolTemplate = template.ExpressionTemplate(fmt.Sprintf(`strconv.FormatBool(%s)`, fbB)).
				Captures(fbB).Imports("strconv").Build()
)

// Replaces `fmt.Sprintf("%t", b)` with `strconv.FormatBool(b)` for better
// performance on bool-to-string conversion.
type PreferStrconvFormatBool struct {
	recipe.Base
}

func (r *PreferStrconvFormatBool) Name() string {
	return "org.openrewrite.golang.codequality.PreferStrconvFormatBool"
}
func (r *PreferStrconvFormatBool) DisplayName() string {
	return "Prefer strconv.FormatBool over fmt.Sprintf"
}
func (r *PreferStrconvFormatBool) Description() string {
	return "Replace `fmt.Sprintf(\"%t\", b)` with `strconv.FormatBool(b)` for better performance."
}
func (r *PreferStrconvFormatBool) Tags() []string { return []string{"performance"} }

func (r *PreferStrconvFormatBool) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{
		{DiagnosticID: "perfsprint", Tool: diagnostic.GolangciLint, HasFix: true},
	}
}

func (r *PreferStrconvFormatBool) Editor() recipe.TreeVisitor {
	return visitor.Init(&preferStrconvFormatBoolVisitor{})
}

type preferStrconvFormatBoolVisitor struct {
	visitor.GoVisitor
}

func (v *preferStrconvFormatBoolVisitor) VisitMethodInvocation(mi *java.MethodInvocation, p any) java.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*java.MethodInvocation)

	match := formatBoolPattern.Match(mi, nil)
	if match == nil {
		return mi
	}

	// Skip unless the argument is a bool, since strconv.FormatBool takes a bool.
	arg, ok := match.GetCapture(fbB).(java.Expression)
	if !ok || !matcher.IsBool(matcher.TypeOfExpression(arg)) {
		return mi
	}

	replaced, ok := formatBoolTemplate.Apply(nil, match).(*java.MethodInvocation)
	if !ok {
		return mi
	}
	recipegolang.MaybeAddImport(v, "strconv", nil, false)
	v.DoAfterVisit(recipe.Service[*recipegolang.ImportService](nil).RemoveUnusedImportsVisitor())
	return replaced.WithPrefix(mi.GetPrefix())
}
