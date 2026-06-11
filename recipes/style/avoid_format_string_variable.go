/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// AvoidFormatStringVariable finds calls like `fmt.Sprintf(variable)` or
// `fmt.Errorf(variable)` where the format string is a variable rather than
// a string literal. This is a potential format string vulnerability if the
// variable contains user input with format directives.
type AvoidFormatStringVariable struct {
	recipe.Base
}

func (r *AvoidFormatStringVariable) Name() string {
	return "org.openrewrite.golang.codequality.AvoidFormatStringVariable"
}
func (r *AvoidFormatStringVariable) DisplayName() string { return "Avoid format string variable" }
func (r *AvoidFormatStringVariable) Description() string {
	return "Find calls like `fmt.Sprintf(variable)` where the format string is not a literal. This is a potential format string vulnerability."
}
func (r *AvoidFormatStringVariable) Tags() []string { return []string{"style", "security"} }

func (r *AvoidFormatStringVariable) Editor() recipe.TreeVisitor {
	return visitor.Init(&avoidFormatStringVariableVisitor{})
}

type avoidFormatStringVariableVisitor struct {
	visitor.GoVisitor
}

func (v *avoidFormatStringVariableVisitor) VisitMethodInvocation(mi *java.MethodInvocation, p any) java.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*java.MethodInvocation)

	if mi.Select == nil {
		return mi
	}

	ident, ok := mi.Select.Element.(*java.Identifier)
	if !ok || ident.Name != "fmt" {
		return mi
	}

	switch mi.Name.Name {
	case "Sprintf", "Errorf", "Printf", "Fprintf":
	default:
		return mi
	}

	// Must have at least one argument (the format string)
	if len(mi.Arguments.Elements) == 0 {
		return mi
	}

	// The first argument must NOT be a string literal
	firstArg := mi.Arguments.Elements[0].Element
	if _, isLiteral := firstArg.(*java.Literal); isLiteral {
		return mi
	}

	mi = mi.WithMarkers(java.MarkupWarn(mi.Markers, "format string is a variable, not a literal; potential format string vulnerability"))
	return mi
}
