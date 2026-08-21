/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling

import (
	"fmt"

	recipegolang "github.com/openrewrite/rewrite/rewrite-go/pkg/recipe/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/template"
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

	tmpl := errorsIsTmpl
	if bin.Operator.Element == java.NotEqual {
		tmpl = notErrorsIsTmpl
	}
	isCall, ok := tmpl.Instantiate(template.NewMatchResult().
		Bind(errorsIsErr, errExpr).
		Bind(errorsIsSentinel, sentinel)).(java.Expression)
	if !ok {
		return bin
	}
	return setExprPrefixLocal(isCall, prefix)
}

var (
	errorsIsErr      = template.Expr("err").WithType("error")
	errorsIsSentinel = template.Expr("sentinel").WithType("error")
	errorsIsTmpl     = template.ExpressionTemplate(fmt.Sprintf("errors.Is(%s, %s)", errorsIsErr, errorsIsSentinel)).
				Captures(errorsIsErr, errorsIsSentinel).
				Imports("errors").
				Build()
	notErrorsIsTmpl = template.ExpressionTemplate(fmt.Sprintf("!errors.Is(%s, %s)", errorsIsErr, errorsIsSentinel)).
			Captures(errorsIsErr, errorsIsSentinel).
			Imports("errors").
			Build()
)

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
