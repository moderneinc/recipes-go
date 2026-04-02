/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// ReduceErrorCheckNesting finds `if err != nil` checks that are nested
// three or more levels deep. Deeply nested error checks are hard to follow
// and often indicate that the function should be refactored.
type ReduceErrorCheckNesting struct {
	recipe.Base
}

func (r *ReduceErrorCheckNesting) Name() string {
	return "org.openrewrite.golang.codequality.ReduceErrorCheckNesting"
}
func (r *ReduceErrorCheckNesting) DisplayName() string {
	return "Reduce error check nesting"
}
func (r *ReduceErrorCheckNesting) Description() string {
	return "Find `if err != nil` checks nested three or more levels deep. Consider refactoring to reduce nesting."
}
func (r *ReduceErrorCheckNesting) Tags() []string { return []string{"style", "lint"} }

func (r *ReduceErrorCheckNesting) Editor() recipe.TreeVisitor {
	return visitor.Init(&reduceErrorCheckNestingVisitor{})
}

type reduceErrorCheckNestingVisitor struct {
	visitor.GoVisitor
	ifDepth int
}

func (v *reduceErrorCheckNestingVisitor) VisitIf(ifStmt *tree.If, p any) tree.J {
	v.ifDepth++
	ifStmt = v.GoVisitor.VisitIf(ifStmt, p).(*tree.If)
	v.ifDepth--

	if v.ifDepth+1 < 3 {
		return ifStmt
	}

	if !isErrNotNil(ifStmt.Condition) {
		return ifStmt
	}

	ifStmt = ifStmt.WithMarkers(
		tree.MarkupWarn(ifStmt.Markers, "deeply nested error check"),
	)
	return ifStmt
}

// isErrNotNil returns true if the expression is `err != nil`.
func isErrNotNil(expr tree.Expression) bool {
	bin, ok := expr.(*tree.Binary)
	if !ok || bin.Operator.Element != tree.NotEqual {
		return false
	}

	leftIdent, leftOk := bin.Left.(*tree.Identifier)
	rightIdent, rightOk := bin.Right.(*tree.Identifier)
	if !leftOk || !rightOk {
		return false
	}
	return leftIdent.Name == "err" && rightIdent.Name == "nil"
}
