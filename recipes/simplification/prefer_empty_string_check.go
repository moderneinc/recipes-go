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

var esS = template.Expr("esS")

var (
	emptyStrEqPattern  = template.Expression(fmt.Sprintf(`len(%s) == 0`, esS)).Captures(esS).Build()
	emptyStrEqTemplate = template.ExpressionTemplate(fmt.Sprintf(`%s == ""`, esS)).Captures(esS).Build()
	emptyStrNePattern  = template.Expression(fmt.Sprintf(`len(%s) != 0`, esS)).Captures(esS).Build()
	emptyStrNeTemplate = template.ExpressionTemplate(fmt.Sprintf(`%s != ""`, esS)).Captures(esS).Build()
)

// Replaces `len(s) == 0` with `s == ""` and `len(s) != 0` with `s != ""`.
type PreferEmptyStringCheck struct {
	recipe.Base
}

func (r *PreferEmptyStringCheck) Name() string {
	return "org.openrewrite.golang.codequality.PreferEmptyStringCheck"
}
func (r *PreferEmptyStringCheck) DisplayName() string {
	return "Prefer empty string check"
}
func (r *PreferEmptyStringCheck) Description() string {
	return "Replace `len(s) == 0` with `s == \"\"` and `len(s) != 0` with `s != \"\"`."
}
func (r *PreferEmptyStringCheck) Tags() []string { return []string{"cleanup", "simplification"} }

func (r *PreferEmptyStringCheck) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{}
}

func (r *PreferEmptyStringCheck) Editor() recipe.TreeVisitor {
	return visitor.Init(&preferEmptyStringCheckVisitor{})
}

type preferEmptyStringCheckVisitor struct {
	visitor.GoVisitor
}

func (v *preferEmptyStringCheckVisitor) VisitBinary(bin *java.Binary, p any) java.J {
	bin = v.GoVisitor.VisitBinary(bin, p).(*java.Binary)

	for _, pt := range []struct {
		pat  *template.GoPattern
		tmpl *template.GoTemplate
	}{
		{emptyStrEqPattern, emptyStrEqTemplate},
		{emptyStrNePattern, emptyStrNeTemplate},
	} {
		match := pt.pat.Match(bin, nil)
		if match == nil {
			continue
		}
		// Skip unless the len() argument is a string, since `== ""` requires a
		// string while len also accepts slices, maps, arrays, and channels.
		arg, ok := match.GetCapture(esS).(java.Expression)
		if !ok || !matcher.IsString(matcher.TypeOfExpression(arg)) {
			return bin
		}
		replaced, ok := pt.tmpl.Apply(nil, match).(*java.Binary)
		if !ok {
			return bin
		}
		return replaced.WithPrefix(bin.GetPrefix())
	}
	return bin
}
