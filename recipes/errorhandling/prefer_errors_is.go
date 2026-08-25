/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling

import (
	"github.com/moderneinc/recipes-go/diagnostic"
	"github.com/moderneinc/recipes-go/recipes/internal/lstutil"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/matcher"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// PreferErrorsIsOverEquality replaces `err == ErrFoo` with `errors.Is(err, ErrFoo)`.
// This is important for wrapped errors where == comparison doesn't check the chain.
type PreferErrorsIsOverEquality struct {
	recipe.Base
}

func (r *PreferErrorsIsOverEquality) Name() string {
	return "org.openrewrite.golang.codequality.PreferErrorsIsOverEquality"
}
func (r *PreferErrorsIsOverEquality) DisplayName() string {
	return "Prefer errors.Is over == for error comparison"
}
func (r *PreferErrorsIsOverEquality) Description() string {
	return "Replace `err == ErrFoo` with `errors.Is(err, ErrFoo)` for correct wrapped error handling."
}
func (r *PreferErrorsIsOverEquality) Tags() []string { return []string{"error-handling"} }

func (r *PreferErrorsIsOverEquality) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{
		{DiagnosticID: "errorlint", Tool: diagnostic.GolangciLint, HasFix: true},
	}
}

func (r *PreferErrorsIsOverEquality) Editor() recipe.TreeVisitor {
	return visitor.Init(&preferErrorsIsVisitor{})
}

type preferErrorsIsVisitor struct {
	visitor.GoVisitor
}

func (v *preferErrorsIsVisitor) VisitBinary(bin *java.Binary, p any) java.J {
	bin = v.GoVisitor.VisitBinary(bin, p).(*java.Binary)

	if bin.Operator.Element != java.Equal && bin.Operator.Element != java.NotEqual {
		return bin
	}

	// The name narrows the candidates; rewriteToErrorsIs has the last word, since
	// it requires both operands to be error values.
	leftIsErr := isErrorSentinel(bin.Left)
	rightIsErr := isErrorSentinel(bin.Right)

	if !leftIsErr && !rightIsErr {
		return bin
	}

	var errExpr, sentinel java.Expression
	if rightIsErr {
		errExpr = bin.Left
		sentinel = bin.Right
	} else {
		errExpr = bin.Right
		sentinel = bin.Left
	}

	// Don't match `err == nil` — that's idiomatic Go
	if isNilIdentifier(sentinel) {
		return bin
	}

	return rewriteToErrorsIs(v, bin, errExpr, sentinel)
}

// Reports whether expr is a value assignable to error, which errors.Is requires
// of both of its arguments and err.Error() requires of its receiver.
func isErrorAssignable(expr java.Expression) bool {
	return matcher.IsAssignableTo(matcher.TypeOfExpression(expr), "error")
}

func isErrorSentinel(expr java.Expression) bool {
	switch n := expr.(type) {
	case *java.Identifier:
		if len(n.Name) >= 3 && n.Name[:3] == "Err" {
			return true
		}
		// Known sentinels
		return n.Name == "EOF"
	case *java.FieldAccess:
		// e.g., io.EOF, os.ErrNotExist
		ident := n.Name.Element
		if len(ident.Name) >= 3 && ident.Name[:3] == "Err" {
			return true
		}
		return ident.Name == "EOF"
	}
	return false
}

func isNilIdentifier(expr java.Expression) bool {
	ident, ok := expr.(*java.Identifier)
	return ok && ident.Name == "nil"
}

// The leading whitespace lives directly on the outermost element, so prefix
// accessors operate on the node's own prefix rather than its leftmost leaf.
func getLeadingPrefixExpr(bin *java.Binary) java.Space {
	return bin.Prefix
}

func stripExprPrefix(expr java.Expression) java.Expression {
	return lstutil.SetExprPrefix(expr, java.Space{})
}
