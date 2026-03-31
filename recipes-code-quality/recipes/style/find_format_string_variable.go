/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindFormatStringVariable finds calls like `fmt.Sprintf(variable)` or
// `fmt.Errorf(variable)` where the format string is a variable rather than
// a string literal. This is a potential format string vulnerability if the
// variable contains user input with format directives.
type FindFormatStringVariable struct {
	recipe.Base
}

func (r *FindFormatStringVariable) Name() string {
	return "org.openrewrite.golang.codequality.FindFormatStringVariable"
}
func (r *FindFormatStringVariable) DisplayName() string { return "Find format string variable" }
func (r *FindFormatStringVariable) Description() string {
	return "Find calls like `fmt.Sprintf(variable)` where the format string is not a literal. This is a potential format string vulnerability."
}
func (r *FindFormatStringVariable) Tags() []string { return []string{"style", "security"} }

func (r *FindFormatStringVariable) Editor() recipe.TreeVisitor {
	return visitor.Init(&findFormatStringVariableVisitor{})
}

type findFormatStringVariableVisitor struct {
	visitor.GoVisitor
}

func (v *findFormatStringVariableVisitor) VisitMethodInvocation(mi *tree.MethodInvocation, p any) tree.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*tree.MethodInvocation)

	if mi.Select == nil {
		return mi
	}

	ident, ok := mi.Select.Element.(*tree.Identifier)
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
	if _, isLiteral := firstArg.(*tree.Literal); isLiteral {
		return mi
	}

	mi = mi.WithMarkers(tree.FoundSearchResult(mi.Markers, "format string is a variable, not a literal; potential format string vulnerability"))
	return mi
}
