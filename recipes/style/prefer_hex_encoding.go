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

var heData = template.Expr("heData")

var (
	hexPattern = template.Expression(fmt.Sprintf(`fmt.Sprintf("%%x", %s)`, heData)).
			Captures(heData).Imports("fmt").Build()
	hexTemplate = template.ExpressionTemplate(fmt.Sprintf(`hex.EncodeToString(%s)`, heData)).
			Captures(heData).Imports("encoding/hex").Build()
)

// Replaces `fmt.Sprintf("%x", data)` with `hex.EncodeToString(data)` for
// clearer intent and better performance.
type PreferHexEncoding struct {
	recipe.Base
}

func (r *PreferHexEncoding) Name() string {
	return "org.openrewrite.golang.codequality.PreferHexEncoding"
}
func (r *PreferHexEncoding) DisplayName() string {
	return "Prefer hex.EncodeToString over fmt.Sprintf"
}
func (r *PreferHexEncoding) Description() string {
	return "Replace `fmt.Sprintf(\"%x\", data)` with `hex.EncodeToString(data)` for clearer intent and better performance."
}
func (r *PreferHexEncoding) Tags() []string { return []string{"style", "cleanup"} }

func (r *PreferHexEncoding) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{
		{DiagnosticID: "perfsprint", Tool: diagnostic.GolangciLint, HasFix: true},
	}
}

func (r *PreferHexEncoding) Editor() recipe.TreeVisitor {
	return visitor.Init(&preferHexEncodingVisitor{})
}

type preferHexEncodingVisitor struct {
	visitor.GoVisitor
}

func (v *preferHexEncodingVisitor) VisitMethodInvocation(mi *java.MethodInvocation, p any) java.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*java.MethodInvocation)

	match := hexPattern.Match(mi, nil)
	if match == nil {
		return mi
	}

	// Skip unless the argument is a []byte, since hex.EncodeToString takes a
	// []byte while %x also accepts a string or an integer.
	arg, ok := match.GetCapture(heData).(java.Expression)
	if !ok || !isByteSlice(matcher.TypeOfExpression(arg)) {
		return mi
	}

	replaced, ok := hexTemplate.Apply(nil, match).(*java.MethodInvocation)
	if !ok {
		return mi
	}
	recipegolang.MaybeAddImport(v, "encoding/hex", nil, false)
	recipegolang.MaybeRemoveImport(v, "fmt")
	return replaced.WithPrefix(mi.GetPrefix())
}

// Reports whether t is a []byte (equivalently []uint8).
func isByteSlice(t java.JavaType) bool {
	switch java.TypeSignature(t) {
	case "byte[]", "uint8[]":
		return true
	}
	return false
}
