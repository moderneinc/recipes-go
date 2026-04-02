/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// WrapErrorWithContext replaces bare `return err` with
// `return fmt.Errorf("funcName: %w", err)`, using the enclosing function name
// as context.
type WrapErrorWithContext struct {
	recipe.Base
}

func (r *WrapErrorWithContext) Name() string {
	return "org.openrewrite.golang.codequality.WrapErrorWithContext"
}
func (r *WrapErrorWithContext) DisplayName() string { return "Wrap error with context" }
func (r *WrapErrorWithContext) Description() string {
	return "Replace bare `return err` with `return fmt.Errorf(\"funcName: %%w\", err)` using the enclosing function name as context."
}
func (r *WrapErrorWithContext) Tags() []string { return []string{"errorhandling", "lint"} }

func (r *WrapErrorWithContext) Editor() recipe.TreeVisitor {
	return visitor.Init(&wrapErrorWithContextVisitor{})
}

type wrapErrorWithContextVisitor struct {
	visitor.GoVisitor
	funcName string
}

func (v *wrapErrorWithContextVisitor) VisitMethodDeclaration(md *tree.MethodDeclaration, p any) tree.J {
	oldName := v.funcName
	if md.Name != nil {
		v.funcName = md.Name.Name
	}
	result := v.GoVisitor.VisitMethodDeclaration(md, p)
	v.funcName = oldName
	return result
}

func (v *wrapErrorWithContextVisitor) VisitReturn(ret *tree.Return, p any) tree.J {
	ret = v.GoVisitor.VisitReturn(ret, p).(*tree.Return)

	// Match: return with a single expression that is an identifier named "err".
	if len(ret.Expressions) != 1 {
		return ret
	}

	expr := ret.Expressions[0].Element
	ident, ok := expr.(*tree.Identifier)
	if !ok || ident.Name != "err" {
		return ret
	}

	// Need an enclosing function name to provide context.
	if v.funcName == "" {
		return ret
	}

	// Build: return fmt.Errorf("funcName: %w", err)
	//
	// AST structure:
	//   MethodInvocation {
	//     Select: "fmt"
	//     Name:   "Errorf"
	//     Args:   [ Literal("funcName: %w"), Identifier("err") ]
	//   }
	fmtIdent := &tree.Identifier{
		Name: "fmt",
	}

	errorfIdent := &tree.Identifier{
		Name: "Errorf",
	}

	formatLit := &tree.Literal{
		Kind:   tree.StringLiteral,
		Source: `"` + v.funcName + `: %w"`,
	}

	errIdent := &tree.Identifier{
		Prefix: tree.SingleSpace,
		Name:   "err",
	}

	errorfCall := &tree.MethodInvocation{
		Prefix: tree.SingleSpace,
		Select: &tree.RightPadded[tree.Expression]{Element: fmtIdent},
		Name:   errorfIdent,
		Arguments: tree.Container[tree.Expression]{
			Elements: []tree.RightPadded[tree.Expression]{
				{Element: formatLit, After: tree.Space{}},
				{Element: errIdent, After: tree.Space{}},
			},
		},
	}

	newExprs := []tree.RightPadded[tree.Expression]{
		{
			Element: errorfCall,
			After:   ret.Expressions[0].After,
			Markers: ret.Expressions[0].Markers,
		},
	}

	c := *ret
	c.Expressions = newExprs
	return &c
}
