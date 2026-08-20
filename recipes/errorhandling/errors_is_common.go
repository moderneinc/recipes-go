/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling

import (
	"github.com/moderneinc/recipes-go/recipes/internal/lstutil"
	recipegolang "github.com/openrewrite/rewrite/rewrite-go/pkg/recipe/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// rewriteToErrorsIs builds `errors.Is(errExpr, sentinel)` (or its negation for a
// `!=` binary) with bin's leading prefix, adding the errors import. It returns
// bin unchanged when either operand is not an error value, since errors.Is
// requires both arguments to be assignable to error.
func rewriteToErrorsIs(v visitor.AfterVisitsProvider, bin *java.Binary, errExpr, sentinel java.Expression) java.J {
	if !isErrorAssignable(errExpr) || !isErrorAssignable(sentinel) {
		return bin
	}

	recipegolang.MaybeAddImport(v, "errors", nil, false)

	// The leading whitespace lives on the outermost element, so carry the
	// binary's prefix onto whichever node ends up outermost.
	prefix := getLeadingPrefixExpr(bin)

	sentinelArg := setExprPrefixLocal(stripExprPrefix(sentinel), java.Space{Whitespace: " "})
	isCall := &java.MethodInvocation{
		Select: &java.RightPadded[java.Expression]{Element: &java.Identifier{Name: "errors", Type: lstutil.NamedType("errors")}},
		Name:   &java.Identifier{Name: "Is"},
		Arguments: java.Container[java.Expression]{
			Elements: []java.RightPadded[java.Expression]{
				{Element: stripExprPrefix(errExpr)},
				{Element: sentinelArg},
			},
		},
		MethodType: lstutil.FuncType("errors", "Is", lstutil.BoolType),
	}

	if bin.Operator.Element == java.NotEqual {
		return &java.Unary{
			Prefix:   prefix,
			Operator: java.LeftPadded[java.UnaryOperator]{Element: java.Not},
			Operand:  isCall,
		}
	}
	return isCall.WithPrefix(prefix)
}

// matchSentinel returns (errExpr, sentinel, true) when one side of bin is the
// package-qualified sentinel `pkg.name` (e.g. io.EOF), with the other side as
// the error expression.
func matchSentinel(bin *java.Binary, pkg, name string) (java.Expression, java.Expression, bool) {
	if isSentinel(bin.Right, pkg, name) {
		return bin.Left, bin.Right, true
	}
	if isSentinel(bin.Left, pkg, name) {
		return bin.Right, bin.Left, true
	}
	return nil, nil, false
}

// isSentinel reports whether expr is the package-qualified value `pkg.name`.
func isSentinel(expr java.Expression, pkg, name string) bool {
	fa, ok := expr.(*java.FieldAccess)
	if !ok {
		return false
	}
	pkgIdent, ok := fa.Target.(*java.Identifier)
	if !ok || pkgIdent.Name != pkg {
		return false
	}
	return fa.Name.Element != nil && fa.Name.Element.Name == name
}
