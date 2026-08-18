/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification

import (
	"fmt"

	"github.com/openrewrite/rewrite/rewrite-go/pkg/matcher"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	recipegolang "github.com/openrewrite/rewrite/rewrite-go/pkg/recipe/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/template"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

var (
	sbwB = template.Expr("sbwB")
	sbwS = template.Expr("sbwS")
)

var (
	sbWriteStringPattern = template.Expression(fmt.Sprintf(`fmt.Fprintf(&%s, "%%s", %s)`, sbwB, sbwS)).
				Captures(sbwB, sbwS).Imports("fmt").Build()
	sbWriteStringTemplate = template.ExpressionTemplate(fmt.Sprintf(`%s.WriteString(%s)`, sbwB, sbwS)).
				Captures(sbwB, sbwS).Build()
)

// Replaces `fmt.Fprintf(&b, "%s", s)` with `b.WriteString(s)` when writing to a
// strings.Builder.
// Staticcheck: S1038
type PreferStringsBuilderWriteString struct {
	recipe.Base
}

func (r *PreferStringsBuilderWriteString) Name() string {
	return "org.openrewrite.golang.codequality.PreferStringsBuilderWriteString"
}
func (r *PreferStringsBuilderWriteString) DisplayName() string {
	return "Prefer strings.Builder WriteString"
}
func (r *PreferStringsBuilderWriteString) Description() string {
	return "Replace `fmt.Fprintf(&b, \"%s\", s)` with `b.WriteString(s)` for more efficient string building."
}
func (r *PreferStringsBuilderWriteString) Tags() []string {
	return []string{"cleanup", "simplification"}
}

func (r *PreferStringsBuilderWriteString) Editor() recipe.TreeVisitor {
	return visitor.Init(&preferStringsBuilderWriteStringVisitor{})
}

type preferStringsBuilderWriteStringVisitor struct {
	visitor.GoVisitor
}

func (v *preferStringsBuilderWriteStringVisitor) VisitMethodInvocation(mi *java.MethodInvocation, p any) java.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*java.MethodInvocation)

	match := sbWriteStringPattern.Match(mi, nil)
	if match == nil {
		return mi
	}

	// Skip unless the formatted value is a string, since Builder.WriteString takes
	// a string while %s also accepts []byte or a fmt.Stringer.
	arg, ok := match.GetCapture(sbwS).(java.Expression)
	if !ok || !matcher.IsString(matcher.TypeOfExpression(arg)) {
		return mi
	}

	replaced, ok := sbWriteStringTemplate.Apply(nil, match).(*java.MethodInvocation)
	if !ok {
		return mi
	}
	recipegolang.MaybeRemoveImport(v, "fmt")
	return replaced.WithPrefix(mi.GetPrefix())
}
