/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// AvoidEmptyInterfaceParam replaces `interface{}` parameter types with `any`
// (Go 1.18+). Parameters already typed as `any` are left unchanged.
type AvoidEmptyInterfaceParam struct {
	recipe.Base
}

func (r *AvoidEmptyInterfaceParam) Name() string {
	return "org.openrewrite.golang.codequality.AvoidEmptyInterfaceParam"
}
func (r *AvoidEmptyInterfaceParam) DisplayName() string { return "Avoid empty interface parameters" }
func (r *AvoidEmptyInterfaceParam) Description() string {
	return "Replace `interface{}` parameter types with `any` (Go 1.18+)."
}
func (r *AvoidEmptyInterfaceParam) Tags() []string { return []string{"style", "lint"} }

func (r *AvoidEmptyInterfaceParam) Editor() recipe.TreeVisitor {
	return visitor.Init(&avoidEmptyInterfaceParamVisitor{})
}

type avoidEmptyInterfaceParamVisitor struct {
	visitor.GoVisitor
}

func (v *avoidEmptyInterfaceParamVisitor) VisitMethodDeclaration(md *tree.MethodDeclaration, p any) tree.J {
	md = v.GoVisitor.VisitMethodDeclaration(md, p).(*tree.MethodDeclaration)

	changed := false
	params := make([]tree.RightPadded[tree.Statement], len(md.Parameters.Elements))
	copy(params, md.Parameters.Elements)

	for i, param := range params {
		vd, ok := param.Element.(*tree.VariableDeclarations)
		if !ok {
			continue
		}
		if isEmptyInterfaceExpr(vd.TypeExpr) {
			// Replace interface{} with any, preserving prefix
			prefix := vd.TypeExpr.(*tree.InterfaceType).Prefix
			newVd := *vd
			newVd.TypeExpr = &tree.Identifier{Prefix: prefix, Name: "any"}
			params[i] = tree.RightPadded[tree.Statement]{
				Element: &newVd,
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

// isEmptyInterfaceExpr returns true if the expression is `interface{}` (an
// InterfaceType with an empty body). It does NOT match `any` — that is already
// the desired form.
func isEmptyInterfaceExpr(expr tree.Expression) bool {
	if expr == nil {
		return false
	}
	it, ok := expr.(*tree.InterfaceType)
	if !ok {
		return false
	}
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
