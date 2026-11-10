/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package redundancy

import (
	"github.com/moderneinc/recipes-go/recipes/internal/lstutil"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// RemoveConstantCondition simplifies conditions only when their constant value
// is represented as a literal. Go's predeclared true/false are identifiers, and
// may be shadowed by variables or constants, even in another source file.
// The current parser does not retain constant-identifier binding, so those
// conditions must remain unchanged until that proof is available.
type RemoveConstantCondition struct{ recipe.Base }

func (r *RemoveConstantCondition) Name() string {
	return "org.openrewrite.golang.codequality.RemoveConstantCondition"
}
func (r *RemoveConstantCondition) DisplayName() string { return "Remove constant if condition" }
func (r *RemoveConstantCondition) Description() string {
	return "Remove constant if conditions only when a boolean literal proves the value. " +
		"Identifier conditions, including true and false, are preserved because Go permits shadowing " +
		"and the current parser does not retain enough binding information to prove them constant."
}
func (r *RemoveConstantCondition) Tags() []string {
	return []string{"cleanup", "redundancy", "RSPEC-S1145"}
}
func (r *RemoveConstantCondition) Editor() recipe.TreeVisitor {
	return visitor.Init(&removeConstantConditionVisitor{})
}

type removeConstantConditionVisitor struct{ visitor.GoVisitor }

func (v *removeConstantConditionVisitor) VisitIf(ifStmt *java.If, p any) java.J {
	ifStmt = v.GoVisitor.VisitIf(ifStmt, p).(*java.If)
	if ifStmt.Condition == nil || lstutil.IsInitWrappedIf(v.Cursor()) || lstutil.HasLabel(ifStmt) {
		return ifStmt
	}
	literal, ok := ifStmt.Condition.Tree.Element.(*java.Literal)
	if !ok {
		return ifStmt
	}
	value, ok := literal.Value.(bool)
	if !ok {
		return ifStmt
	}
	if value {
		if block, ok := ifStmt.ThenPart.Element.(*java.Block); ok {
			return block.WithPrefix(ifStmt.Prefix)
		}
		return ifStmt
	}
	if ifStmt.ElsePart != nil {
		if block, ok := ifStmt.ElsePart.Body.Element.(*java.Block); ok {
			return block.WithPrefix(ifStmt.Prefix)
		}
		// Do not discard an else-if chain whose body cannot be preserved here.
		return ifStmt
	}
	return &java.Empty{}
}
