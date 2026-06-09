/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	recipegolang "github.com/openrewrite/rewrite/rewrite-go/pkg/recipe/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
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

func (v *wrapErrorWithContextVisitor) VisitMethodDeclaration(md *java.MethodDeclaration, p any) java.J {
	oldName := v.funcName
	if md.Name != nil {
		v.funcName = md.Name.Name
	}
	result := v.GoVisitor.VisitMethodDeclaration(md, p)
	v.funcName = oldName
	return result
}

func (v *wrapErrorWithContextVisitor) VisitReturn(ret *java.Return, p any) java.J {
	ret = v.GoVisitor.VisitReturn(ret, p).(*java.Return)

	// Match: return with a single expression that is an identifier named "err".
	if ret.Expression == nil {
		return ret
	}

	ident, ok := ret.Expression.(*java.Identifier)
	if !ok || ident.Name != "err" {
		return ret
	}

	// Need an enclosing function name to provide context.
	if v.funcName == "" {
		return ret
	}

	// The rewrite introduces a reference to the `fmt` package; ensure it is imported.
	recipegolang.MaybeAddImport(v, "fmt", nil, false)

	// Build: return fmt.Errorf("funcName: %w", err)
	//
	// AST structure:
	//   MethodInvocation {
	//     Select: "fmt"
	//     Name:   "Errorf"
	//     Args:   [ Literal("funcName: %w"), Identifier("err") ]
	//   }
	fmtIdent := &java.Identifier{
		Name: "fmt",
	}

	errorfIdent := &java.Identifier{
		Name: "Errorf",
	}

	formatLit := &java.Literal{
		Source: `"` + v.funcName + `: %w"`,
	}

	errIdent := &java.Identifier{
		Prefix: java.SingleSpace,
		Name:   "err",
	}

	errorfCall := &java.MethodInvocation{
		Prefix: java.SingleSpace,
		Select: &java.RightPadded[java.Expression]{Element: fmtIdent},
		Name:   errorfIdent,
		Arguments: java.Container[java.Expression]{
			Elements: []java.RightPadded[java.Expression]{
				{Element: formatLit, After: java.Space{}},
				{Element: errIdent, After: java.Space{}},
			},
		},
	}

	c := *ret
	c.Expression = errorfCall
	return &c
}
