/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package naming

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// UseCtxForContextParam renames function parameters of type context.Context that
// are not named "ctx" to "ctx", and renames all usages of the old name in the
// function body. Go convention is to always name context parameters "ctx".
type UseCtxForContextParam struct {
	recipe.Base
}

func (r *UseCtxForContextParam) Name() string {
	return "org.openrewrite.golang.codequality.UseCtxForContextParam"
}
func (r *UseCtxForContextParam) DisplayName() string {
	return "Use ctx for context.Context parameter"
}
func (r *UseCtxForContextParam) Description() string {
	return "Rename function parameters of type context.Context that are not named \"ctx\" to \"ctx\", including all usages in the function body."
}
func (r *UseCtxForContextParam) Tags() []string { return []string{"naming"} }

func (r *UseCtxForContextParam) Editor() recipe.TreeVisitor {
	return visitor.Init(&useCtxForContextParamVisitor{})
}

type useCtxForContextParamVisitor struct {
	visitor.GoVisitor
}

func (v *useCtxForContextParamVisitor) VisitMethodDeclaration(md *tree.MethodDeclaration, p any) tree.J {
	md = v.GoVisitor.VisitMethodDeclaration(md, p).(*tree.MethodDeclaration)

	if md.Name == nil {
		return md
	}

	// Collect old names of context params that need renaming.
	var oldNames []string
	params := make([]tree.RightPadded[tree.Statement], len(md.Parameters.Elements))
	copy(params, md.Parameters.Elements)
	changed := false

	for i, paramRP := range params {
		vd, ok := paramRP.Element.(*tree.VariableDeclarations)
		if !ok {
			continue
		}

		// Check if the type is context.Context (FieldAccess: Target=context, Name=Context).
		if !isContextType(vd.TypeExpr) {
			continue
		}

		renamedVD, oldName := v.renameContextParam(vd)
		if oldName != "" {
			oldNames = append(oldNames, oldName)
			params[i] = tree.RightPadded[tree.Statement]{
				Element: renamedVD,
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

	// Rename all usages of the old names in the function body.
	if c.Body != nil && len(oldNames) > 0 {
		renamer := visitor.Init(&ctxRenameVisitor{oldNames: oldNames})
		result := renamer.Visit(c.Body, p)
		c.Body = result.(*tree.Block)
	}

	return &c
}

// renameContextParam renames the context parameter to "ctx" and returns the old name.
// Returns the original vd and "" if no rename is needed.
func (v *useCtxForContextParamVisitor) renameContextParam(vd *tree.VariableDeclarations) (*tree.VariableDeclarations, string) {
	vars := make([]tree.RightPadded[*tree.VariableDeclarator], len(vd.Variables))
	copy(vars, vd.Variables)
	changed := false
	oldName := ""

	for j, varRP := range vars {
		decl := varRP.Element
		if decl.Name == nil {
			continue
		}

		if decl.Name.Name == "ctx" {
			continue
		}

		oldName = decl.Name.Name
		renamed := decl.WithName(decl.Name.WithName("ctx"))
		vars[j] = tree.RightPadded[*tree.VariableDeclarator]{
			Element: renamed,
			After:   varRP.After,
			Markers: varRP.Markers,
		}
		changed = true
	}

	if !changed {
		return vd, ""
	}
	c := *vd
	c.Variables = vars
	return &c, oldName
}

// ctxRenameVisitor renames identifiers matching the old context param names to "ctx".
type ctxRenameVisitor struct {
	visitor.GoVisitor
	oldNames []string
}

func (v *ctxRenameVisitor) VisitIdentifier(ident *tree.Identifier, p any) tree.J {
	ident = v.GoVisitor.VisitIdentifier(ident, p).(*tree.Identifier)
	for _, oldName := range v.oldNames {
		if ident.Name == oldName {
			return ident.WithName("ctx")
		}
	}
	return ident
}

// isContextType checks if a type expression is context.Context (a FieldAccess
// where Target is "context" and Name is "Context").
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
