/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package redundancy

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindConstantCondition finds `if true { ... }` or `if false { ... }` where
// the condition is a boolean literal. A constant condition makes the if
// statement either dead code or unconditionally executed.
type FindConstantCondition struct {
	recipe.Base
}

func (r *FindConstantCondition) Name() string {
	return "org.openrewrite.golang.codequality.FindConstantCondition"
}
func (r *FindConstantCondition) DisplayName() string { return "Find constant if condition" }
func (r *FindConstantCondition) Description() string {
	return "Find `if true { ... }` or `if false { ... }` where the condition is always true or false."
}
func (r *FindConstantCondition) Tags() []string { return []string{"cleanup", "redundancy"} }

func (r *FindConstantCondition) Editor() recipe.TreeVisitor {
	return visitor.Init(&findConstantConditionVisitor{})
}

type findConstantConditionVisitor struct {
	visitor.GoVisitor
}

func (v *findConstantConditionVisitor) VisitIf(ifStmt *tree.If, p any) tree.J {
	ifStmt = v.GoVisitor.VisitIf(ifStmt, p).(*tree.If)

	// Condition must be an Identifier named "true" or "false".
	ident, ok := ifStmt.Condition.(*tree.Identifier)
	if !ok {
		return ifStmt
	}
	if ident.Name != "true" && ident.Name != "false" {
		return ifStmt
	}

	ifStmt = ifStmt.WithMarkers(
		tree.FoundSearchResult(ifStmt.Markers, "constant condition: "+ident.Name),
	)
	return ifStmt
}
