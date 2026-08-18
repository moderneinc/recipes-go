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

var snrS = template.Expr("snrS")

// PreferStringsNewReader replaces `bytes.NewReader([]byte(s))` with
// `strings.NewReader(s)`. When the source is already a string, converting
// it to []byte only to wrap it in a bytes.Reader is wasteful; strings.NewReader
// avoids the allocation.
// Staticcheck: S1036
type PreferStringsNewReader struct {
	recipe.Base
}

func (r *PreferStringsNewReader) Name() string {
	return "org.openrewrite.golang.codequality.PreferStringsNewReader"
}
func (r *PreferStringsNewReader) DisplayName() string { return "Prefer strings.NewReader" }
func (r *PreferStringsNewReader) Description() string {
	return "Replace `bytes.NewReader([]byte(s))` with `strings.NewReader(s)` to avoid an unnecessary string-to-byte-slice conversion."
}
func (r *PreferStringsNewReader) Tags() []string { return []string{"cleanup", "simplification"} }

func (r *PreferStringsNewReader) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{
		{DiagnosticID: "mirror", Tool: diagnostic.GolangciLint, HasFix: true},
	}
}

var (
	newReaderPattern = template.Expression(fmt.Sprintf(`bytes.NewReader([]byte(%s))`, snrS)).
				Captures(snrS).Imports("bytes").Build()
	newReaderTemplate = template.ExpressionTemplate(fmt.Sprintf(`strings.NewReader(%s)`, snrS)).
				Captures(snrS).Imports("strings").Build()
)

func (r *PreferStringsNewReader) Editor() recipe.TreeVisitor {
	return visitor.Init(&preferStringsNewReaderVisitor{})
}

type preferStringsNewReaderVisitor struct {
	visitor.GoVisitor
}

func (v *preferStringsNewReaderVisitor) VisitMethodInvocation(mi *java.MethodInvocation, p any) java.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*java.MethodInvocation)

	match := newReaderPattern.Match(mi, nil)
	if match == nil {
		return mi
	}

	// Skip unless the converted value is a string, since strings.NewReader takes
	// a string and would not compile on a []byte argument.
	inner, ok := match.GetCapture(snrS).(java.Expression)
	if !ok || !matcher.IsString(matcher.TypeOfExpression(inner)) {
		return mi
	}

	// Skip when the result is required as *bytes.Reader by a return or a typed
	// variable declaration, where strings.NewReader's *strings.Reader would not
	// compile, but leave an interface target such as io.Reader to rewrite.
	if t, ok := requiredResultType(v.Cursor()); ok && t == "*bytes.Reader" {
		return mi
	}

	replaced := newReaderTemplate.Apply(nil, match)
	if replaced == nil {
		return mi
	}
	newCall, ok := replaced.(*java.MethodInvocation)
	if !ok {
		return mi
	}

	recipegolang.MaybeAddImport(v, "strings", nil, false)
	recipegolang.MaybeRemoveImport(v, "bytes")
	return newCall.WithPrefix(mi.GetPrefix())
}
