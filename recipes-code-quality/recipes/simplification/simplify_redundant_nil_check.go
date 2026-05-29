/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification

import (
	"github.com/moderneinc/recipes-go/recipes-code-quality/diagnostic"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// SimplifyRedundantNilCheck simplifies `if x != nil && len(x) > 0` to
// `if len(x) > 0` for slices and maps, since len of nil returns 0.
// Staticcheck: S1009
type SimplifyRedundantNilCheck struct {
	recipe.Base
}

func (r *SimplifyRedundantNilCheck) Name() string {
	return "org.openrewrite.golang.codequality.SimplifyRedundantNilCheck"
}
func (r *SimplifyRedundantNilCheck) DisplayName() string { return "Simplify redundant nil check" }
func (r *SimplifyRedundantNilCheck) Description() string {
	return "Simplify `x != nil && len(x) > 0` to `len(x) > 0` for slices and maps."
}
func (r *SimplifyRedundantNilCheck) Tags() []string { return []string{"cleanup", "simplification"} }

func (r *SimplifyRedundantNilCheck) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{
		{DiagnosticID: "S1009", Tool: diagnostic.Staticcheck, HasFix: true},
	}
}

func (r *SimplifyRedundantNilCheck) Editor() recipe.TreeVisitor {
	return visitor.Init(&simplifyRedundantNilCheckVisitor{})
}

type simplifyRedundantNilCheckVisitor struct {
	visitor.GoVisitor
}

func (v *simplifyRedundantNilCheckVisitor) VisitBinary(bin *java.Binary, p any) java.J {
	bin = v.GoVisitor.VisitBinary(bin, p).(*java.Binary)

	if bin.Operator.Element != java.LogicalAnd {
		return bin
	}

	// Check pattern: x != nil && len(x) > 0  (or len(x) > 0 && x != nil)
	leftNilCheck, leftVar := isNilNotEqualCheck(bin.Left)
	rightNilCheck, rightVar := isNilNotEqualCheck(bin.Right)

	if leftNilCheck && isLenCheck(bin.Right, leftVar) {
		// x != nil && len(x) > 0  ->  len(x) > 0
		// Preserve the outer binary's leading prefix (space after "if")
		return setLeadingPrefix(bin.Right, leadingPrefix(bin))
	}
	if rightNilCheck && isLenCheck(bin.Left, rightVar) {
		// len(x) > 0 && x != nil  ->  len(x) > 0
		return setLeadingPrefix(bin.Left, leadingPrefix(bin))
	}

	return bin
}

// isNilNotEqualCheck checks if the expression is `x != nil` or `nil != x`.
// Returns true and the variable name if it matches.
func isNilNotEqualCheck(expr java.Expression) (bool, string) {
	bin, ok := expr.(*java.Binary)
	if !ok || bin.Operator.Element != java.NotEqual {
		return false, ""
	}

	leftNil := isNilIdent(bin.Left)
	rightNil := isNilIdent(bin.Right)

	if rightNil {
		if name := identName(bin.Left); name != "" {
			return true, name
		}
	}
	if leftNil {
		if name := identName(bin.Right); name != "" {
			return true, name
		}
	}
	return false, ""
}

func isNilIdent(expr java.Expression) bool {
	ident, ok := expr.(*java.Identifier)
	return ok && ident.Name == "nil"
}

func identName(expr java.Expression) string {
	ident, ok := expr.(*java.Identifier)
	if ok {
		return ident.Name
	}
	return ""
}

// isLenCheck checks if the expression is `len(varName) > 0` or `len(varName) != 0`
// or `len(varName) >= 1`.
func isLenCheck(expr java.Expression, varName string) bool {
	bin, ok := expr.(*java.Binary)
	if !ok {
		return false
	}

	// Check left side is len(varName)
	mi, ok := bin.Left.(*java.MethodInvocation)
	if !ok || mi.Select != nil || mi.Name.Name != "len" {
		return false
	}
	args := mi.Arguments.Elements
	if len(args) == 0 {
		return false
	}
	// Find the actual argument (skip Empty sentinels)
	for _, arg := range args {
		if ident, ok := arg.Element.(*java.Identifier); ok {
			if ident.Name == varName {
				return isPositiveComparison(bin.Operator.Element, bin.Right)
			}
		}
	}
	return false
}

// setLeadingPrefix sets the prefix on the leftmost leaf of an expression,
// which is where the effective leading whitespace lives in the Go LST.
func setLeadingPrefix(expr java.Expression, prefix java.Space) java.Expression {
	switch n := expr.(type) {
	case *java.Binary:
		return n.WithLeft(setLeadingPrefix(n.Left, prefix))
	case *java.Identifier:
		return n.WithPrefix(prefix)
	case *java.Literal:
		return n.WithPrefix(prefix)
	case *java.MethodInvocation:
		if n.Select != nil {
			sel := *n.Select
			sel.Element = setLeadingPrefix(sel.Element, prefix)
			n.Select = &sel
			return n
		}
		return n.WithName(n.Name.WithPrefix(prefix))
	default:
		return expr
	}
}

// isPositiveComparison checks if the operator and right operand form a
// "length is positive" check: > 0, != 0, >= 1.
func isPositiveComparison(op java.BinaryOperator, right java.Expression) bool {
	lit, ok := right.(*java.Literal)
	if !ok {
		return false
	}
	switch op {
	case java.GreaterThan:
		return lit.Source == "0"
	case java.NotEqual:
		return lit.Source == "0"
	case java.GreaterThanOrEqual:
		return lit.Source == "1"
	default:
		return false
	}
}
