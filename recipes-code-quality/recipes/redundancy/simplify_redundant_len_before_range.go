/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package redundancy

import (
	"strings"

	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// SimplifyRedundantLenBeforeRange removes the redundant `if len(s) > 0` wrapper
// around `for ... range s` loops. Range over a nil or empty slice/map produces
// zero iterations, so the len check is unnecessary.
type SimplifyRedundantLenBeforeRange struct {
	recipe.Base
}

func (r *SimplifyRedundantLenBeforeRange) Name() string {
	return "org.openrewrite.golang.codequality.SimplifyRedundantLenBeforeRange"
}
func (r *SimplifyRedundantLenBeforeRange) DisplayName() string {
	return "Simplify redundant len check before range"
}
func (r *SimplifyRedundantLenBeforeRange) Description() string {
	return "Remove `if len(s) > 0 { for ... range s }` where the len check is redundant because range over nil/empty produces zero iterations."
}
func (r *SimplifyRedundantLenBeforeRange) Tags() []string {
	return []string{"cleanup", "redundancy"}
}

func (r *SimplifyRedundantLenBeforeRange) Editor() recipe.TreeVisitor {
	return visitor.Init(&simplifyRedundantLenBeforeRangeVisitor{})
}

type simplifyRedundantLenBeforeRangeVisitor struct {
	visitor.GoVisitor
}

func (v *simplifyRedundantLenBeforeRangeVisitor) VisitIf(ifStmt *java.If, p any) java.J {
	ifStmt = v.GoVisitor.VisitIf(ifStmt, p).(*java.If)

	// Must not have an else clause.
	if ifStmt.ElsePart != nil {
		return ifStmt
	}

	// Must not have an init statement.
	if ifStmt.Init != nil {
		return ifStmt
	}

	// The then block must have exactly one statement.
	if ifStmt.Then == nil || len(ifStmt.Then.Statements) != 1 {
		return ifStmt
	}

	// That single statement must be a ForEachLoop (range).
	forEach, ok := ifStmt.Then.Statements[0].Element.(*java.ForEachLoop)
	if !ok {
		return ifStmt
	}

	// Condition must be `len(x) > 0` or `len(x) != 0` or `len(x) >= 1`.
	varName := lenCheckVarName(ifStmt.Condition)
	if varName == "" {
		return ifStmt
	}

	// The range iterable must be over the same variable.
	rangeVar := forEachIterableName(forEach)
	if rangeVar != varName {
		return ifStmt
	}

	// Replace the if statement with the for-range loop, preserving the if's prefix.
	// Dedent the for-range body by one tab since it's being lifted out of the if block.
	dedent := visitor.Init(&dedentVisitor{})
	result := dedent.Visit(forEach, p)
	return result.(*java.ForEachLoop).WithPrefix(ifStmt.Prefix)
}

// lenCheckVarName extracts the variable name from a `len(x) > 0` style condition.
// Returns "" if the condition does not match.
func lenCheckVarName(cond java.Expression) string {
	bin, ok := cond.(*java.Binary)
	if !ok {
		return ""
	}

	// Left side must be len(x)
	mi, ok := bin.Left.(*java.MethodInvocation)
	if !ok || mi.Select != nil || mi.Name.Name != "len" {
		return ""
	}

	// Must have exactly one real argument.
	var argName string
	for _, arg := range mi.Arguments.Elements {
		if ident, ok := arg.Element.(*java.Identifier); ok {
			argName = ident.Name
			break
		}
	}
	if argName == "" {
		return ""
	}

	// Right side must be a literal 0 or 1 with the right operator.
	lit, ok := bin.Right.(*java.Literal)
	if !ok {
		return ""
	}

	switch bin.Operator.Element {
	case java.GreaterThan:
		if lit.Source == "0" {
			return argName
		}
	case java.NotEqual:
		if lit.Source == "0" {
			return argName
		}
	case java.GreaterThanOrEqual:
		if lit.Source == "1" {
			return argName
		}
	}
	return ""
}

// forEachIterableName extracts the identifier name from the iterable
// of a ForEachLoop. Returns "" if the iterable is not a simple identifier.
func forEachIterableName(forEach *java.ForEachLoop) string {
	ident, ok := forEach.Control.Iterable.(*java.Identifier)
	if !ok {
		return ""
	}
	return ident.Name
}

// dedentVisitor removes one tab from every whitespace in a subtree.
type dedentVisitor struct {
	visitor.GoVisitor
}

func (v *dedentVisitor) VisitSpace(space java.Space, p any) java.Space {
	if strings.Contains(space.Whitespace, "\t") {
		space.Whitespace = strings.Replace(space.Whitespace, "\t", "", 1)
	}
	return space
}
