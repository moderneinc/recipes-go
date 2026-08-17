/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

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

var sqS = template.Expr("sqS")

var (
	quotePattern = template.Expression(fmt.Sprintf(`fmt.Sprintf("%%q", %s)`, sqS)).
			Captures(sqS).Imports("fmt").Build()
	quoteTemplate = template.ExpressionTemplate(fmt.Sprintf(`strconv.Quote(%s)`, sqS)).
			Captures(sqS).Imports("strconv").Build()
)

// Replaces `fmt.Sprintf("%q", s)` with `strconv.Quote(s)` for clearer intent
// when quoting strings.
type PreferStrconvQuote struct {
	recipe.Base
}

func (r *PreferStrconvQuote) Name() string {
	return "org.openrewrite.golang.codequality.PreferStrconvQuote"
}
func (r *PreferStrconvQuote) DisplayName() string {
	return "Prefer strconv.Quote over fmt.Sprintf"
}
func (r *PreferStrconvQuote) Description() string {
	return "Replace `fmt.Sprintf(\"%q\", s)` with `strconv.Quote(s)` for clearer intent when quoting strings."
}
func (r *PreferStrconvQuote) Tags() []string { return []string{"style", "cleanup"} }

func (r *PreferStrconvQuote) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{
		{DiagnosticID: "perfsprint", Tool: diagnostic.GolangciLint, HasFix: true},
	}
}

func (r *PreferStrconvQuote) Editor() recipe.TreeVisitor {
	return visitor.Init(&preferStrconvQuoteVisitor{})
}

type preferStrconvQuoteVisitor struct {
	visitor.GoVisitor
}

func (v *preferStrconvQuoteVisitor) VisitMethodInvocation(mi *java.MethodInvocation, p any) java.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*java.MethodInvocation)

	match := quotePattern.Match(mi, nil)
	if match == nil {
		return mi
	}

	// Skip unless the argument is a string, since strconv.Quote takes a string
	// while %q also accepts a rune or []byte.
	arg, ok := match.GetCapture(sqS).(java.Expression)
	if !ok || !matcher.IsString(matcher.TypeOfExpression(arg)) {
		return mi
	}

	replaced, ok := quoteTemplate.Apply(nil, match).(*java.MethodInvocation)
	if !ok {
		return mi
	}
	recipegolang.MaybeAddImport(v, "strconv", nil, false)
	v.DoAfterVisit(recipe.Service[*recipegolang.ImportService](nil).RemoveUnusedImportsVisitor())
	return replaced.WithPrefix(mi.GetPrefix())
}
