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
	wsW   = template.Expr("wsW")
	wsStr = template.Expr("wsStr")
)

var (
	ioWriteStringPattern = template.Expression(fmt.Sprintf(`fmt.Fprintf(%s, "%%s", %s)`, wsW, wsStr)).
				Captures(wsW, wsStr).Imports("fmt").Build()
	ioWriteStringTemplate = template.ExpressionTemplate(fmt.Sprintf(`io.WriteString(%s, %s)`, wsW, wsStr)).
				Captures(wsW, wsStr).Imports("io").Build()
)

// Replaces `fmt.Fprintf(w, "%s", s)` with `io.WriteString(w, s)`.
// Staticcheck: S1025
type PreferIoWriteString struct {
	recipe.Base
}

func (r *PreferIoWriteString) Name() string {
	return "org.openrewrite.golang.codequality.PreferIoWriteString"
}
func (r *PreferIoWriteString) DisplayName() string { return "Prefer io.WriteString" }
func (r *PreferIoWriteString) Description() string {
	return "Replace `fmt.Fprintf(w, \"%s\", s)` with `io.WriteString(w, s)`."
}
func (r *PreferIoWriteString) Tags() []string { return []string{"cleanup", "simplification"} }

func (r *PreferIoWriteString) Editor() recipe.TreeVisitor {
	return visitor.Init(&preferIoWriteStringVisitor{})
}

type preferIoWriteStringVisitor struct {
	visitor.GoVisitor
}

func (v *preferIoWriteStringVisitor) VisitMethodInvocation(mi *java.MethodInvocation, p any) java.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*java.MethodInvocation)

	match := ioWriteStringPattern.Match(mi, nil)
	if match == nil {
		return mi
	}

	// Skip unless the formatted value is a string, since io.WriteString takes a
	// string while %s also accepts []byte or a fmt.Stringer.
	arg, ok := match.GetCapture(wsStr).(java.Expression)
	if !ok || !matcher.IsString(matcher.TypeOfExpression(arg)) {
		return mi
	}

	replaced, ok := ioWriteStringTemplate.Apply(nil, match).(*java.MethodInvocation)
	if !ok {
		return mi
	}
	recipegolang.MaybeAddImport(v, "io", nil, false)
	recipegolang.MaybeRemoveImport(v, "fmt")
	return replaced.WithPrefix(mi.GetPrefix())
}
