/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling

import (
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

	// Check if one side looks like an error sentinel (Err* identifier or io.EOF etc.)
	// and the other side is a variable (likely an err variable).
	// Without type attribution, we match: any comparison where one side starts with "Err"
	// or is a known error sentinel like io.EOF.
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

	// Build errors.Is(err, sentinel) or !errors.Is(err, sentinel)
	prefix := getLeadingPrefixExpr(bin)

	errorsIdent := &java.Identifier{Prefix: prefix, Name: "errors"}
	isIdent := &java.Identifier{Name: "Is"}

	errArg := stripExprPrefix(errExpr)
	sentinelArg := stripExprPrefix(sentinel)
	// Add space before second argument (after comma)
	sentinelArgWithSpace := setExprPrefixLocal(sentinelArg, java.Space{Whitespace: " "})

	isCall := &java.MethodInvocation{
		Select: &java.RightPadded[java.Expression]{Element: errorsIdent},
		Name:   isIdent,
		Arguments: java.Container[java.Expression]{
			Elements: []java.RightPadded[java.Expression]{
				{Element: errArg},
				{Element: sentinelArgWithSpace},
			},
		},
	}

	if bin.Operator.Element == java.NotEqual {
		return &java.Unary{
			Prefix:   prefix,
			Operator: java.LeftPadded[java.UnaryOperator]{Element: java.Not},
			Operand:  setMethodInvocationPrefix(isCall, java.Space{}),
		}
	}
	return isCall
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

func getLeadingPrefixExpr(bin *java.Binary) java.Space {
	return getExprPrefix(bin.Left)
}

func getExprPrefix(expr java.Expression) java.Space {
	switch n := expr.(type) {
	case *java.Identifier:
		return n.Prefix
	case *java.Literal:
		return n.Prefix
	case *java.FieldAccess:
		return getExprPrefix(n.Target)
	case *java.MethodInvocation:
		if n.Select != nil {
			return getExprPrefix(n.Select.Element)
		}
		return getExprPrefix(n.Name)
	default:
		return java.Space{}
	}
}

func stripExprPrefix(expr java.Expression) java.Expression {
	switch n := expr.(type) {
	case *java.Identifier:
		return n.WithPrefix(java.Space{})
	case *java.Literal:
		return n.WithPrefix(java.Space{})
	case *java.FieldAccess:
		return n.WithTarget(stripExprPrefix(n.Target))
	default:
		return expr
	}
}

func setMethodInvocationPrefix(mi *java.MethodInvocation, prefix java.Space) java.Expression {
	if mi.Select != nil {
		sel := *mi.Select
		sel.Element = setExprPrefixLocal(sel.Element, prefix)
		return &java.MethodInvocation{
			ID: mi.ID, Prefix: mi.Prefix, Markers: mi.Markers,
			Select: &sel, Name: mi.Name, Arguments: mi.Arguments, MethodType: mi.MethodType,
		}
	}
	return mi.WithPrefix(prefix)
}

func setExprPrefixLocal(expr java.Expression, prefix java.Space) java.Expression {
	switch n := expr.(type) {
	case *java.Identifier:
		return n.WithPrefix(prefix)
	case *java.Literal:
		return n.WithPrefix(prefix)
	case *java.FieldAccess:
		return n.WithTarget(setExprPrefixLocal(n.Target, prefix))
	default:
		return expr
	}
}
