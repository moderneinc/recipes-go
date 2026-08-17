/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling

import (
	"github.com/moderneinc/recipes-go/diagnostic"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/matcher"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	recipegolang "github.com/openrewrite/rewrite/rewrite-go/pkg/recipe/golang"
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

	// errors.Is requires both operands to be error values; skip a comparison that
	// only matched by the Err* name, such as an int constant named ErrLevel.
	if !isErrorAssignable(errExpr) || !isErrorAssignable(sentinel) {
		return bin
	}

	// The rewrite introduces a reference to the `errors` package; ensure it is imported.
	recipegolang.MaybeAddImport(v, "errors", nil, false)

	// Build errors.Is(err, sentinel) or !errors.Is(err, sentinel). The leading
	// whitespace lives on the outermost element, so carry the binary's prefix
	// onto whichever node ends up outermost (the call, or the negating unary).
	prefix := getLeadingPrefixExpr(bin)

	errorsIdent := &java.Identifier{Name: "errors"}
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
			Operand:  isCall,
		}
	}
	return isCall.WithPrefix(prefix)
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
	return setExprPrefixLocal(expr, java.Space{})
}

func setExprPrefixLocal(expr java.Expression, prefix java.Space) java.Expression {
	switch n := expr.(type) {
	case *java.Identifier:
		return n.WithPrefix(prefix)
	case *java.Literal:
		return n.WithPrefix(prefix)
	case *java.FieldAccess:
		return n.WithPrefix(prefix)
	case *java.MethodInvocation:
		return n.WithPrefix(prefix)
	default:
		return expr
	}
}
