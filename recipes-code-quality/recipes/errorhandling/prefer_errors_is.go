/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
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

func (v *preferErrorsIsVisitor) VisitBinary(bin *tree.Binary, p any) tree.J {
	bin = v.GoVisitor.VisitBinary(bin, p).(*tree.Binary)

	if bin.Operator.Element != tree.Equal && bin.Operator.Element != tree.NotEqual {
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

	var errExpr, sentinel tree.Expression
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

	errorsIdent := &tree.Identifier{Prefix: prefix, Name: "errors"}
	isIdent := &tree.Identifier{Name: "Is"}

	errArg := stripExprPrefix(errExpr)
	sentinelArg := stripExprPrefix(sentinel)
	// Add space before second argument (after comma)
	sentinelArgWithSpace := setExprPrefixLocal(sentinelArg, tree.Space{Whitespace: " "})

	isCall := &tree.MethodInvocation{
		Select: &tree.RightPadded[tree.Expression]{Element: errorsIdent},
		Name:   isIdent,
		Arguments: tree.Container[tree.Expression]{
			Elements: []tree.RightPadded[tree.Expression]{
				{Element: errArg},
				{Element: sentinelArgWithSpace},
			},
		},
	}

	if bin.Operator.Element == tree.NotEqual {
		return &tree.Unary{
			Prefix:   prefix,
			Operator: tree.LeftPadded[tree.UnaryOperator]{Element: tree.Not},
			Operand:  setMethodInvocationPrefix(isCall, tree.Space{}),
		}
	}
	return isCall
}

func isErrorSentinel(expr tree.Expression) bool {
	switch n := expr.(type) {
	case *tree.Identifier:
		if len(n.Name) >= 3 && n.Name[:3] == "Err" {
			return true
		}
		// Known sentinels
		return n.Name == "EOF"
	case *tree.FieldAccess:
		// e.g., io.EOF, os.ErrNotExist
		ident := n.Name.Element
		if len(ident.Name) >= 3 && ident.Name[:3] == "Err" {
			return true
		}
		return ident.Name == "EOF"
	}
	return false
}

func isNilIdentifier(expr tree.Expression) bool {
	ident, ok := expr.(*tree.Identifier)
	return ok && ident.Name == "nil"
}

func getLeadingPrefixExpr(bin *tree.Binary) tree.Space {
	return getExprPrefix(bin.Left)
}

func getExprPrefix(expr tree.Expression) tree.Space {
	switch n := expr.(type) {
	case *tree.Identifier:
		return n.Prefix
	case *tree.Literal:
		return n.Prefix
	case *tree.FieldAccess:
		return getExprPrefix(n.Target)
	case *tree.MethodInvocation:
		if n.Select != nil {
			return getExprPrefix(n.Select.Element)
		}
		return getExprPrefix(n.Name)
	default:
		return tree.Space{}
	}
}

func stripExprPrefix(expr tree.Expression) tree.Expression {
	switch n := expr.(type) {
	case *tree.Identifier:
		return n.WithPrefix(tree.Space{})
	case *tree.Literal:
		return n.WithPrefix(tree.Space{})
	case *tree.FieldAccess:
		return n.WithTarget(stripExprPrefix(n.Target))
	default:
		return expr
	}
}

func setMethodInvocationPrefix(mi *tree.MethodInvocation, prefix tree.Space) tree.Expression {
	if mi.Select != nil {
		sel := *mi.Select
		sel.Element = setExprPrefixLocal(sel.Element, prefix)
		return &tree.MethodInvocation{
			ID: mi.ID, Prefix: mi.Prefix, Markers: mi.Markers,
			Select: &sel, Name: mi.Name, Arguments: mi.Arguments, MethodType: mi.MethodType,
		}
	}
	return mi.WithPrefix(prefix)
}

func setExprPrefixLocal(expr tree.Expression, prefix tree.Space) tree.Expression {
	switch n := expr.(type) {
	case *tree.Identifier:
		return n.WithPrefix(prefix)
	case *tree.Literal:
		return n.WithPrefix(prefix)
	case *tree.FieldAccess:
		return n.WithTarget(setExprPrefixLocal(n.Target, prefix))
	default:
		return expr
	}
}
