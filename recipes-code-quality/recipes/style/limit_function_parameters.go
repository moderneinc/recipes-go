/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// LimitFunctionParameters finds functions with more than 5 parameters.
// Too many parameters suggest the function should accept a struct instead.
type LimitFunctionParameters struct {
	recipe.Base
}

func (r *LimitFunctionParameters) Name() string {
	return "org.openrewrite.golang.codequality.LimitFunctionParameters"
}
func (r *LimitFunctionParameters) DisplayName() string {
	return "Limit function parameters"
}
func (r *LimitFunctionParameters) Description() string {
	return "Find functions with more than 5 parameters. Consider grouping parameters into a struct."
}
func (r *LimitFunctionParameters) Tags() []string { return []string{"style", "lint"} }

func (r *LimitFunctionParameters) Editor() recipe.TreeVisitor {
	return visitor.Init(&limitFunctionParametersVisitor{})
}

type limitFunctionParametersVisitor struct {
	visitor.GoVisitor
}

func (v *limitFunctionParametersVisitor) VisitMethodDeclaration(md *java.MethodDeclaration, p any) java.J {
	md = v.GoVisitor.VisitMethodDeclaration(md, p).(*java.MethodDeclaration)

	if md.Name == nil {
		return md
	}

	count := 0
	for _, param := range md.Parameters.Elements {
		if _, isEmpty := param.Element.(*java.Empty); !isEmpty {
			count++
		}
	}

	if count <= 5 {
		return md
	}

	md = md.WithName(md.Name.WithMarkers(
		java.MarkupInfo(md.Name.Markers, "function has too many parameters"),
	))
	return md
}
