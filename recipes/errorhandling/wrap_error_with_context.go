/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling

import (
	"fmt"

	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	recipegolang "github.com/openrewrite/rewrite/rewrite-go/pkg/recipe/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/template"
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

	// fmt.Errorf returns error, so only wrap when the function returns a single
	// error result; a concrete error type such as *MyErr would not accept it.
	if !enclosingReturnsSingleError(v.Cursor()) {
		return ret
	}

	// The rewrite introduces a reference to the `fmt` package; ensure it is imported.
	recipegolang.MaybeAddImport(v, "fmt", nil, false)

	// Reuse the wrapped identifier so the argument keeps its error type.
	errorfCall := wrapErrorTmpl.Instantiate(template.NewMatchResult().
		Bind(wrapErrorFormat, &java.Literal{Source: `"` + v.funcName + `: %w"`}).
		Bind(wrapErrorErr, ident))
	if errorfCall == nil {
		return ret
	}

	c := *ret
	c.Expression = errorfCall.(*java.MethodInvocation).WithPrefix(java.SingleSpace)
	return &c
}

var (
	wrapErrorFormat = template.Expr("format").WithType("string")
	wrapErrorErr    = template.Expr("err").WithType("error")
	wrapErrorTmpl   = template.ExpressionTemplate(fmt.Sprintf("fmt.Errorf(%s, %s)", wrapErrorFormat, wrapErrorErr)).
			Captures(wrapErrorFormat, wrapErrorErr).
			Imports("fmt").
			Build()
)
