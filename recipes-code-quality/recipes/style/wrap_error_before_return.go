/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// WrapErrorBeforeReturn replaces `return nil, err` with
// `return nil, fmt.Errorf("funcName: %w", err)`, using the enclosing function
// name as context. Wrapping errors makes debugging easier.
type WrapErrorBeforeReturn struct {
	recipe.Base
}

func (r *WrapErrorBeforeReturn) Name() string {
	return "org.openrewrite.golang.codequality.WrapErrorBeforeReturn"
}
func (r *WrapErrorBeforeReturn) DisplayName() string { return "Wrap error before return" }
func (r *WrapErrorBeforeReturn) Description() string {
	return "Replace `return nil, err` with `return nil, fmt.Errorf(\"funcName: %%w\", err)` using the enclosing function name as context."
}
func (r *WrapErrorBeforeReturn) Tags() []string { return []string{"style", "errorhandling"} }

func (r *WrapErrorBeforeReturn) Editor() recipe.TreeVisitor {
	return visitor.Init(&wrapErrorBeforeReturnVisitor{})
}

type wrapErrorBeforeReturnVisitor struct {
	visitor.GoVisitor
	funcName string
}

func (v *wrapErrorBeforeReturnVisitor) VisitMethodDeclaration(md *tree.MethodDeclaration, p any) tree.J {
	oldName := v.funcName
	if md.Name != nil {
		v.funcName = md.Name.Name
	}
	result := v.GoVisitor.VisitMethodDeclaration(md, p)
	v.funcName = oldName
	return result
}

func (v *wrapErrorBeforeReturnVisitor) VisitReturn(ret *tree.Return, p any) tree.J {
	ret = v.GoVisitor.VisitReturn(ret, p).(*tree.Return)

	if len(ret.Expressions) < 2 {
		return ret
	}

	// First expression must be the nil identifier.
	firstIdent, firstOk := ret.Expressions[0].Element.(*tree.Identifier)
	if !firstOk || firstIdent.Name != "nil" {
		return ret
	}

	// Last expression must be the bare "err" identifier.
	lastIdx := len(ret.Expressions) - 1
	lastIdent, lastOk := ret.Expressions[lastIdx].Element.(*tree.Identifier)
	if !lastOk || lastIdent.Name != "err" {
		return ret
	}

	// Need an enclosing function name to provide context.
	if v.funcName == "" {
		return ret
	}

	// Build: fmt.Errorf("funcName: %w", err)
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
		Prefix: lastIdent.Prefix,
		Select: &tree.RightPadded[tree.Expression]{Element: fmtIdent},
		Name:   errorfIdent,
		Arguments: tree.Container[tree.Expression]{
			Elements: []tree.RightPadded[tree.Expression]{
				{Element: formatLit},
				{Element: errIdent},
			},
		},
	}

	// Replace the last expression (bare err) with the fmt.Errorf call.
	newExprs := make([]tree.RightPadded[tree.Expression], len(ret.Expressions))
	copy(newExprs, ret.Expressions)
	newExprs[lastIdx] = tree.RightPadded[tree.Expression]{
		Element: errorfCall,
		After:   ret.Expressions[lastIdx].After,
		Markers: ret.Expressions[lastIdx].Markers,
	}

	c := *ret
	c.Expressions = newExprs
	return &c
}
