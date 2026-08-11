/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification

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

var (
	scA = template.Expr("scA")
	scB = template.Expr("scB")
)

var (
	sprintfConcatPattern = template.Expression(fmt.Sprintf(`fmt.Sprintf("%%s%%s", %s, %s)`, scA, scB)).
				Captures(scA, scB).Imports("fmt").Build()
	sprintfConcatTemplate = template.ExpressionTemplate(fmt.Sprintf(`%s + %s`, scA, scB)).
				Captures(scA, scB).Build()
)

// Replaces `fmt.Sprintf("%s%s", a, b)` with `a + b` to avoid unnecessary
// formatting overhead for simple string concatenation.
type SimplifySprintfConcat struct {
	recipe.Base
}

func (r *SimplifySprintfConcat) Name() string {
	return "org.openrewrite.golang.codequality.SimplifySprintfConcat"
}
func (r *SimplifySprintfConcat) DisplayName() string { return "Simplify fmt.Sprintf string concat" }
func (r *SimplifySprintfConcat) Description() string {
	return "Replace `fmt.Sprintf(\"%s%s\", a, b)` with `a + b` for simple string concatenation."
}
func (r *SimplifySprintfConcat) Tags() []string { return []string{"cleanup", "simplification"} }

func (r *SimplifySprintfConcat) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{}
}

func (r *SimplifySprintfConcat) Editor() recipe.TreeVisitor {
	return visitor.Init(&simplifySprintfConcatVisitor{})
}

type simplifySprintfConcatVisitor struct {
	visitor.GoVisitor
}

func (v *simplifySprintfConcatVisitor) VisitMethodInvocation(mi *java.MethodInvocation, p any) java.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*java.MethodInvocation)

	match := sprintfConcatPattern.Match(mi, nil)
	if match == nil {
		return mi
	}

	// Skip unless both arguments are strings, since `+` concatenates strings but
	// %s also accepts []byte and fmt.Stringer, which cannot be added.
	a, aok := match.GetCapture(scA).(java.Expression)
	b, bok := match.GetCapture(scB).(java.Expression)
	if !aok || !bok ||
		!matcher.IsString(matcher.TypeOfExpression(a)) ||
		!matcher.IsString(matcher.TypeOfExpression(b)) {
		return mi
	}

	replaced, ok := sprintfConcatTemplate.Apply(nil, match).(*java.Binary)
	if !ok {
		return mi
	}
	v.DoAfterVisit(recipe.Service[*recipegolang.ImportService](nil).RemoveUnusedImportsVisitor())
	return replaced.WithPrefix(mi.GetPrefix())
}
