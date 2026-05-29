/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package redundancy

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/printer"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// RemoveDuplicateConditions removes else-if branches whose condition already
// appeared earlier in the same if/else-if chain. The duplicated branch is dead
// code because the first matching condition will always execute instead.
//
//	if x > 0 { a() } else if x > 0 { b() } else { c() }
//	// becomes
//	if x > 0 { a() } else { c() }
type RemoveDuplicateConditions struct {
	recipe.Base
}

func (r *RemoveDuplicateConditions) Name() string {
	return "org.openrewrite.golang.codequality.RemoveDuplicateConditions"
}
func (r *RemoveDuplicateConditions) DisplayName() string { return "Remove duplicate conditions" }
func (r *RemoveDuplicateConditions) Description() string {
	return "Remove else-if branches whose condition duplicates an earlier branch in the " +
		"same if/else-if chain, since the later branch is dead code."
}
func (r *RemoveDuplicateConditions) Tags() []string {
	return []string{"cleanup", "redundancy", "RSPEC-S1862"}
}

func (r *RemoveDuplicateConditions) Editor() recipe.TreeVisitor {
	return visitor.Init(&removeDuplicateConditionsVisitor{})
}

type removeDuplicateConditionsVisitor struct {
	visitor.GoVisitor
}

func (v *removeDuplicateConditionsVisitor) VisitIf(ifStmt *java.If, p any) java.J {
	ifStmt = v.GoVisitor.VisitIf(ifStmt, p).(*java.If)

	// Only process the outermost if in a chain (don't re-enter from inner else-if).
	// We'll walk the full chain ourselves.
	result := removeDuplicateBranches(ifStmt)
	if result == nil {
		return ifStmt
	}
	return result
}

// removeDuplicateBranches walks an if/else-if chain, collects conditions, and
// removes any else-if whose condition has already appeared. Returns nil if
// nothing changed.
func removeDuplicateBranches(ifStmt *java.If) *java.If {
	// Collect the flat list of conditions we've seen so far.
	seen := []string{printCondition(ifStmt.Condition)}
	changed := false

	current := ifStmt
	for current.ElsePart != nil {
		elseIf, ok := current.ElsePart.Element.(*java.If)
		if !ok {
			// Plain else { } — end of chain.
			break
		}

		condStr := printCondition(elseIf.Condition)
		if containsStr(seen, condStr) {
			// This else-if duplicates an earlier condition — remove it.
			// Splice: current's else becomes the duplicate's else.
			current.ElsePart = elseIf.ElsePart
			changed = true
			// Don't advance current — re-check the new else part.
			continue
		}
		seen = append(seen, condStr)
		current = elseIf
	}

	if !changed {
		return nil
	}
	return ifStmt
}

func printCondition(expr java.Expression) string {
	return printer.Print(setCondPrefix(expr, java.Space{}))
}

func containsStr(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

// setCondPrefix sets the leading whitespace prefix on an expression.
func setCondPrefix(expr java.Expression, prefix java.Space) java.Expression {
	switch n := expr.(type) {
	case *java.Identifier:
		return n.WithPrefix(prefix)
	case *java.Literal:
		return n.WithPrefix(prefix)
	case *java.Parentheses:
		return n.WithPrefix(prefix)
	case *java.Binary:
		return &java.Binary{
			ID: n.ID, Prefix: n.Prefix, Markers: n.Markers,
			Left: setCondPrefix(n.Left, prefix), Operator: n.Operator, Right: n.Right, Type: n.Type,
		}
	case *java.Unary:
		return &java.Unary{
			ID: n.ID, Prefix: prefix, Markers: n.Markers,
			Operator: n.Operator, Operand: n.Operand, Type: n.Type,
		}
	case *java.FieldAccess:
		return n.WithPrefix(prefix)
	case *java.MethodInvocation:
		return n.WithPrefix(prefix)
	default:
		return expr
	}
}
