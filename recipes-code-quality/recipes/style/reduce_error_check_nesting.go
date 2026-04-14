/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// ReduceErrorCheckNesting applies the guard-clause refactoring to
// `if err == nil { body }` by inverting the condition to
// `if err != nil { return err }` followed by the body statements.
// This reduces nesting in error-handling code.
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
	return "Invert `if err == nil { body }` to `if err != nil { return err }` followed by the body, reducing nesting in error-handling code."
}
func (r *ReduceErrorCheckNesting) Tags() []string { return []string{"style", "lint"} }

func (r *ReduceErrorCheckNesting) Editor() recipe.TreeVisitor {
	return visitor.Init(&reduceErrorCheckNestingVisitor{})
}

type reduceErrorCheckNestingVisitor struct {
	visitor.GoVisitor
}

func (v *reduceErrorCheckNestingVisitor) VisitBlock(block *tree.Block, p any) tree.J {
	block = v.GoVisitor.VisitBlock(block, p).(*tree.Block)

	changed := false
	var newStmts []tree.RightPadded[tree.Statement]

	dedent := visitor.Init(&nestingDedentVisitor{})

	for _, rp := range block.Statements {
		ifStmt, ok := rp.Element.(*tree.If)
		if !ok || ifStmt.Init != nil || ifStmt.ElsePart != nil || ifStmt.Then == nil {
			newStmts = append(newStmts, rp)
			continue
		}

		if !isErrEqualNil(ifStmt.Condition) {
			newStmts = append(newStmts, rp)
			continue
		}

		changed = true

		// Build `if err != nil { return err }`
		errReturn := []tree.RightPadded[tree.Expression]{
			{Element: &tree.Identifier{Prefix: tree.SingleSpace, Name: "err"}},
		}
		guard := buildErrGuard(ifStmt, errReturn)
		newStmts = append(newStmts, tree.RightPadded[tree.Statement]{Element: guard})

		// Splice the body statements out, dedented by one level.
		for _, bodyRP := range ifStmt.Then.Statements {
			bodyDedented := dedent.Visit(bodyRP.Element, nil).(tree.Statement)
			newStmts = append(newStmts, tree.RightPadded[tree.Statement]{
				Element: bodyDedented,
				After:   bodyRP.After,
				Markers: bodyRP.Markers,
			})
		}
	}

	if !changed {
		return block
	}
	return block.WithStatements(newStmts)
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
