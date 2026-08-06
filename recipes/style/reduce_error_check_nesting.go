/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/moderneinc/recipes-go/recipes/internal/lstutil"
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

	// Only rewrite the function's top-level body, and only when it returns a
	// single error, so the synthesized `return err` compiles and an early return
	// from a nested block does not change control flow.
	if !lstutil.IsFunctionBodyBlock(v.Cursor()) {
		return block
	}
	md, ok := visitor.FirstEnclosing[*java.MethodDeclaration](v.Cursor())
	if !ok || !funcReturnsError(md) {
		return block
	}

	changed := false
	var newStmts []java.RightPadded[java.Statement]

	dedent := visitor.Init(&nestingDedentVisitor{})

	for i, rp := range block.Statements {
		// An `if init; cond` is a golang.StatementWithInit, not a *java.If, so the
		// assertion already excludes init-bearing ifs.
		ifStmt, ok := rp.Element.(*java.If)
		if !ok || ifStmt.ElsePart != nil || ifStmt.Condition == nil {
			newStmts = append(newStmts, rp)
			continue
		}
		thenBlock, ok := ifStmt.ThenPart.Element.(*java.Block)
		if !ok {
			newStmts = append(newStmts, rp)
			continue
		}

		if !isErrEqualNil(ifStmt.Condition.Tree.Element) {
			newStmts = append(newStmts, rp)
			continue
		}

		// Inverting to an early return is behaviour-preserving only when the `if`
		// is the block's last statement.
		if !isLastRealStatement(block.Statements, i) {
			newStmts = append(newStmts, rp)
			continue
		}

		changed = true

		// Build `if err != nil { return err }`
		errReturn := &java.Identifier{Prefix: java.SingleSpace, Name: "err"}
		guard := buildErrGuard(ifStmt, errReturn)
		newStmts = append(newStmts, java.RightPadded[java.Statement]{Element: guard})

		// Splice the body statements out, dedented by one level.
		for _, bodyRP := range thenBlock.Statements {
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
