/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package naming

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindContextParamNotCtx finds function parameters of type context.Context that
// are not named "ctx". Go convention is to always name context parameters "ctx".
type FindContextParamNotCtx struct {
	recipe.Base
}

func (r *FindContextParamNotCtx) Name() string {
	return "org.openrewrite.golang.codequality.FindContextParamNotCtx"
}
func (r *FindContextParamNotCtx) DisplayName() string {
	return "Find context.Context parameter not named ctx"
}
func (r *FindContextParamNotCtx) Description() string {
	return "Find function parameters of type context.Context that are not named \"ctx\". Go convention is to always name context parameters \"ctx\"."
}
func (r *FindContextParamNotCtx) Tags() []string { return []string{"naming"} }

func (r *FindContextParamNotCtx) Editor() recipe.TreeVisitor {
	return visitor.Init(&findContextParamNotCtxVisitor{})
}

type findContextParamNotCtxVisitor struct {
	visitor.GoVisitor
}

func (v *findContextParamNotCtxVisitor) VisitMethodDeclaration(md *tree.MethodDeclaration, p any) tree.J {
	md = v.GoVisitor.VisitMethodDeclaration(md, p).(*tree.MethodDeclaration)

	if md.Name == nil {
		return md
	}

	changed := false
	params := make([]tree.RightPadded[tree.Statement], len(md.Parameters.Elements))
	copy(params, md.Parameters.Elements)

	for i, paramRP := range params {
		vd, ok := paramRP.Element.(*tree.VariableDeclarations)
		if !ok {
			continue
		}

		// Check if the type is context.Context (FieldAccess: Target=context, Name=Context).
		if !isContextType(vd.TypeExpr) {
			continue
		}

		markedVD := v.checkContextParam(vd)
		if markedVD != vd {
			params[i] = tree.RightPadded[tree.Statement]{
				Element: markedVD,
				After:   paramRP.After,
				Markers: paramRP.Markers,
			}
			changed = true
		}
	}

	if !changed {
		return md
	}

	c := *md
	c.Parameters = tree.Container[tree.Statement]{
		Before:   md.Parameters.Before,
		Elements: params,
		Markers:  md.Parameters.Markers,
	}
	return &c
}

// isContextType checks whether an expression represents the context.Context type.
func isContextType(expr tree.Expression) bool {
	fa, ok := expr.(*tree.FieldAccess)
	if !ok {
		return false
	}
	target, ok := fa.Target.(*tree.Identifier)
	if !ok {
		return false
	}
	return target.Name == "context" && fa.Name.Element.Name == "Context"
}

// checkContextParam checks the variable declarators in a parameter declaration
// and marks any that are not named "ctx".
func (v *findContextParamNotCtxVisitor) checkContextParam(vd *tree.VariableDeclarations) *tree.VariableDeclarations {
	changed := false
	vars := make([]tree.RightPadded[*tree.VariableDeclarator], len(vd.Variables))
	copy(vars, vd.Variables)

	for j, varRP := range vars {
		decl := varRP.Element
		if decl.Name == nil {
			continue
		}

		if decl.Name.Name == "ctx" {
			continue
		}

		marked := decl.WithName(decl.Name.WithMarkers(
			tree.FoundSearchResult(decl.Name.Markers, "context.Context parameter should be named ctx"),
		))
		vars[j] = tree.RightPadded[*tree.VariableDeclarator]{
			Element: marked,
			After:   varRP.After,
			Markers: varRP.Markers,
		}
		changed = true
	}

	if !changed {
		return vd
	}
	c := *vd
	c.Variables = vars
	return &c
}
