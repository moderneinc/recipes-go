/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// LimitReturnValues finds functions with more than 3 return values.
// Too many return values make the call site unwieldy; consider returning
// a struct instead.
type LimitReturnValues struct {
	recipe.Base
}

func (r *LimitReturnValues) Name() string {
	return "org.openrewrite.golang.codequality.LimitReturnValues"
}
func (r *LimitReturnValues) DisplayName() string { return "Limit return values" }
func (r *LimitReturnValues) Description() string {
	return "Find functions with more than 3 return values. Consider returning a struct instead."
}
func (r *LimitReturnValues) Tags() []string { return []string{"style", "lint"} }

func (r *LimitReturnValues) Editor() recipe.TreeVisitor {
	return visitor.Init(&limitReturnValuesVisitor{})
}

type limitReturnValuesVisitor struct {
	visitor.GoVisitor
}

func (v *limitReturnValuesVisitor) VisitMethodDeclaration(md *java.MethodDeclaration, p any) java.J {
	md = v.GoVisitor.VisitMethodDeclaration(md, p).(*java.MethodDeclaration)

	if md.Name == nil || md.ReturnType == nil {
		return md
	}

	tl, ok := md.ReturnType.(*golang.TypeList)
	if !ok {
		// Single return value -- not a problem.
		return md
	}

	count := 0
	for _, elem := range tl.Types.Elements {
		if _, isEmpty := elem.Element.(*java.Empty); !isEmpty {
			count++
		}
	}

	if count <= 3 {
		return md
	}

	md = md.WithName(md.Name.WithMarkers(
		java.MarkupInfo(md.Name.Markers, "function has too many return values"),
	))
	return md
}
