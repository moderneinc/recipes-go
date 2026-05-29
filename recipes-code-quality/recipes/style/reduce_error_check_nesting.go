/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
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

func (v *reduceErrorCheckNestingVisitor) VisitBlock(block *java.Block, p any) java.J {
	block = v.GoVisitor.VisitBlock(block, p).(*java.Block)

	changed := false
	var newStmts []java.RightPadded[java.Statement]

	dedent := visitor.Init(&nestingDedentVisitor{})

	for _, rp := range block.Statements {
		ifStmt, ok := rp.Element.(*java.If)
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
		errReturn := []java.RightPadded[java.Expression]{
			{Element: &java.Identifier{Prefix: java.SingleSpace, Name: "err"}},
		}
		guard := buildErrGuard(ifStmt, errReturn)
		newStmts = append(newStmts, java.RightPadded[java.Statement]{Element: guard})

		// Splice the body statements out, dedented by one level.
		for _, bodyRP := range ifStmt.Then.Statements {
			bodyDedented := dedent.Visit(bodyRP.Element, nil).(java.Statement)
			newStmts = append(newStmts, java.RightPadded[java.Statement]{
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
func isErrNotNil(expr java.Expression) bool {
	bin, ok := expr.(*java.Binary)
	if !ok || bin.Operator.Element != java.NotEqual {
		return false
	}

	leftIdent, leftOk := bin.Left.(*java.Identifier)
	rightIdent, rightOk := bin.Right.(*java.Identifier)
	if !leftOk || !rightOk {
		return false
	}
	return leftIdent.Name == "err" && rightIdent.Name == "nil"
}
