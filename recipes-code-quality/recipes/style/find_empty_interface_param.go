/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindEmptyInterfaceParam finds function parameters typed as `interface{}` or
// `any`. Using empty interfaces loses type safety and should be replaced with
// concrete types or constrained interfaces where possible.
type FindEmptyInterfaceParam struct {
	recipe.Base
}

func (r *FindEmptyInterfaceParam) Name() string {
	return "org.openrewrite.golang.codequality.FindEmptyInterfaceParam"
}
func (r *FindEmptyInterfaceParam) DisplayName() string { return "Find empty interface parameters" }
func (r *FindEmptyInterfaceParam) Description() string {
	return "Find function parameters typed as `interface{}` or `any`. Empty interfaces lose type safety."
}
func (r *FindEmptyInterfaceParam) Tags() []string { return []string{"style", "lint"} }

func (r *FindEmptyInterfaceParam) Editor() recipe.TreeVisitor {
	return visitor.Init(&findEmptyInterfaceParamVisitor{})
}

type findEmptyInterfaceParamVisitor struct {
	visitor.GoVisitor
}

func (v *findEmptyInterfaceParamVisitor) VisitMethodDeclaration(md *tree.MethodDeclaration, p any) tree.J {
	md = v.GoVisitor.VisitMethodDeclaration(md, p).(*tree.MethodDeclaration)

	changed := false
	params := make([]tree.RightPadded[tree.Statement], len(md.Parameters.Elements))
	copy(params, md.Parameters.Elements)

	for i, param := range params {
		vd, ok := param.Element.(*tree.VariableDeclarations)
		if !ok {
			continue
		}
		if isEmptyInterfaceType(vd.TypeExpr) {
			marked := vd.WithMarkers(tree.FoundSearchResult(vd.Markers, "avoid empty interface parameter"))
			params[i] = tree.RightPadded[tree.Statement]{
				Element: marked,
				After:   param.After,
				Markers: param.Markers,
			}
			changed = true
		}
	}

	if !changed {
		return md
	}

	newParams := md.Parameters
	newParams.Elements = params
	c := *md
	c.Parameters = newParams
	return &c
}

// isEmptyInterfaceType returns true if the expression is `interface{}` (an
// InterfaceType with an empty body) or the predeclared identifier `any`.
func isEmptyInterfaceType(expr tree.Expression) bool {
	if expr == nil {
		return false
	}
	// Check for `interface{}` — an InterfaceType with no real statements.
	if it, ok := expr.(*tree.InterfaceType); ok {
		if it.Body == nil {
			return true
		}
		for _, s := range it.Body.Statements {
			if _, isEmpty := s.Element.(*tree.Empty); !isEmpty {
				return false
			}
		}
		return true
	}
	// Check for the predeclared identifier `any`.
	if ident, ok := expr.(*tree.Identifier); ok {
		return ident.Name == "any"
	}
	return false
}
