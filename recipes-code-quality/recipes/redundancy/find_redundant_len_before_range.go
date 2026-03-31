/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package redundancy

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindRedundantLenBeforeRange finds `if len(s) > 0 { for _, v := range s { ... } }`
// patterns where the len check is redundant because range over a nil or empty
// slice/map produces zero iterations.
type FindRedundantLenBeforeRange struct {
	recipe.Base
}

func (r *FindRedundantLenBeforeRange) Name() string {
	return "org.openrewrite.golang.codequality.FindRedundantLenBeforeRange"
}
func (r *FindRedundantLenBeforeRange) DisplayName() string {
	return "Find redundant len check before range"
}
func (r *FindRedundantLenBeforeRange) Description() string {
	return "Find `if len(s) > 0 { for ... range s }` where the len check is redundant because range over nil/empty produces zero iterations."
}
func (r *FindRedundantLenBeforeRange) Tags() []string {
	return []string{"cleanup", "redundancy"}
}

func (r *FindRedundantLenBeforeRange) Editor() recipe.TreeVisitor {
	return visitor.Init(&findRedundantLenBeforeRangeVisitor{})
}

type findRedundantLenBeforeRangeVisitor struct {
	visitor.GoVisitor
}

func (v *findRedundantLenBeforeRangeVisitor) VisitIf(ifStmt *tree.If, p any) tree.J {
	ifStmt = v.GoVisitor.VisitIf(ifStmt, p).(*tree.If)

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
	forEach, ok := ifStmt.Then.Statements[0].Element.(*tree.ForEachLoop)
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

	ifStmt = ifStmt.WithMarkers(
		tree.FoundSearchResult(ifStmt.Markers, "len check before range is redundant"),
	)
	return ifStmt
}

// lenCheckVarName extracts the variable name from a `len(x) > 0` style condition.
// Returns "" if the condition does not match.
func lenCheckVarName(cond tree.Expression) string {
	bin, ok := cond.(*tree.Binary)
	if !ok {
		return ""
	}

	// Left side must be len(x)
	mi, ok := bin.Left.(*tree.MethodInvocation)
	if !ok || mi.Select != nil || mi.Name.Name != "len" {
		return ""
	}

	// Must have exactly one real argument.
	var argName string
	for _, arg := range mi.Arguments.Elements {
		if ident, ok := arg.Element.(*tree.Identifier); ok {
			argName = ident.Name
			break
		}
	}
	if argName == "" {
		return ""
	}

	// Right side must be a literal 0 or 1 with the right operator.
	lit, ok := bin.Right.(*tree.Literal)
	if !ok {
		return ""
	}

	switch bin.Operator.Element {
	case tree.GreaterThan:
		if lit.Source == "0" {
			return argName
		}
	case tree.NotEqual:
		if lit.Source == "0" {
			return argName
		}
	case tree.GreaterThanOrEqual:
		if lit.Source == "1" {
			return argName
		}
	}
	return ""
}

// forEachIterableName extracts the identifier name from the iterable
// of a ForEachLoop. Returns "" if the iterable is not a simple identifier.
func forEachIterableName(forEach *tree.ForEachLoop) string {
	ident, ok := forEach.Control.Iterable.(*tree.Identifier)
	if !ok {
		return ""
	}
	return ident.Name
}
