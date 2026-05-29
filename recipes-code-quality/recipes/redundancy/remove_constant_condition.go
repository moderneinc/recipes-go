/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package redundancy

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// RemoveConstantCondition removes `if true { ... }` and `if false { ... }`
// where the condition is a boolean literal. For `if true`, the body is inlined;
// for `if false`, the dead code is removed (or the else body is kept).
type RemoveConstantCondition struct {
	recipe.Base
}

func (r *RemoveConstantCondition) Name() string {
	return "org.openrewrite.golang.codequality.RemoveConstantCondition"
}
func (r *RemoveConstantCondition) DisplayName() string { return "Remove constant if condition" }
func (r *RemoveConstantCondition) Description() string {
	return "Remove `if true { ... }` (inline body) and `if false { ... }` (remove dead code)."
}
func (r *RemoveConstantCondition) Tags() []string { return []string{"cleanup", "redundancy"} }

func (r *RemoveConstantCondition) Editor() recipe.TreeVisitor {
	return visitor.Init(&removeConstantConditionVisitor{})
}

type removeConstantConditionVisitor struct {
	visitor.GoVisitor
}

func (v *removeConstantConditionVisitor) VisitIf(ifStmt *java.If, p any) java.J {
	ifStmt = v.GoVisitor.VisitIf(ifStmt, p).(*java.If)

	// Condition must be an Identifier named "true" or "false".
	ident, ok := ifStmt.Condition.(*java.Identifier)
	if !ok {
		return ifStmt
	}

	if ident.Name == "true" {
		// Replace `if true { body }` with the body block (preserving the if's prefix).
		return ifStmt.Then.WithPrefix(ifStmt.Prefix)
	}

	if ident.Name == "false" {
		// `if false { } else { elseBody }` — keep the else body.
		if ifStmt.ElsePart != nil {
			if block, ok := ifStmt.ElsePart.Element.(*java.Block); ok {
				return block.WithPrefix(ifStmt.Prefix)
			}
		}
		// `if false { body }` — remove dead code.
		return &java.Empty{}
	}

	return ifStmt
}
