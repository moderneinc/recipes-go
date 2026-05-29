/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"strings"

	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// ReduceNestingDepth applies the guard-clause refactoring to
// `if err == nil { body }` by inverting the condition to
// `if err != nil { return }` followed by the body statements.
// This reduces nesting by one level.
// golangci-lint: nestif
type ReduceNestingDepth struct {
	recipe.Base
}

func (r *ReduceNestingDepth) Name() string {
	return "org.openrewrite.golang.codequality.ReduceNestingDepth"
}
func (r *ReduceNestingDepth) DisplayName() string { return "Reduce nesting depth" }
func (r *ReduceNestingDepth) Description() string {
	return "Invert `if err == nil { body }` to `if err != nil { return }` followed by the body, reducing nesting by one level."
}
func (r *ReduceNestingDepth) Tags() []string { return []string{"style", "lint"} }

func (r *ReduceNestingDepth) Editor() recipe.TreeVisitor {
	return visitor.Init(&reduceNestingDepthVisitor{})
}

type reduceNestingDepthVisitor struct {
	visitor.GoVisitor
}

func (v *reduceNestingDepthVisitor) VisitBlock(block *java.Block, p any) java.J {
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

		// Build `if err != nil { return }`
		guard := buildErrGuard(ifStmt, nil)
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

// isErrEqualNil returns true if the expression is `err == nil`.
func isErrEqualNil(expr java.Expression) bool {
	bin, ok := expr.(*java.Binary)
	if !ok || bin.Operator.Element != java.Equal {
		return false
	}

	leftIdent, leftOk := bin.Left.(*java.Identifier)
	rightIdent, rightOk := bin.Right.(*java.Identifier)
	if !leftOk || !rightOk {
		return false
	}
	return leftIdent.Name == "err" && rightIdent.Name == "nil"
}

// buildErrGuard constructs `if err != nil { return }` or `if err != nil { return err }`,
// reusing the prefix of the original if statement.
// When returnExpr is non-nil it is used as the return value.
func buildErrGuard(ifStmt *java.If, returnExpr []java.RightPadded[java.Expression]) *java.If {
	// Build `err != nil` from the original `err == nil` condition.
	origBin := ifStmt.Condition.(*java.Binary)
	invertedCond := &java.Binary{
		Prefix:   origBin.Prefix,
		Left:     origBin.Left,
		Operator: java.LeftPadded[java.BinaryOperator]{Before: origBin.Operator.Before, Element: java.NotEqual},
		Right:    origBin.Right,
	}

	ret := &java.Return{
		Prefix:      java.Space{Whitespace: "\n" + guardIndent(ifStmt.Prefix)},
		Expressions: returnExpr,
	}

	guardBody := &java.Block{
		Prefix: java.SingleSpace,
		Statements: []java.RightPadded[java.Statement]{
			{Element: ret},
		},
		End: java.Space{Whitespace: "\n" + baseIndent(ifStmt.Prefix)},
	}

	return &java.If{
		Prefix:    ifStmt.Prefix,
		Condition: invertedCond,
		Then:      guardBody,
	}
}

// baseIndent extracts the indentation (everything after the last newline)
// from a Space's Whitespace field.
func baseIndent(space java.Space) string {
	ws := space.Whitespace
	if idx := strings.LastIndex(ws, "\n"); idx >= 0 {
		return ws[idx+1:]
	}
	return ws
}

// guardIndent returns one extra tab level of indentation for the guard body.
func guardIndent(space java.Space) string {
	return baseIndent(space) + "\t"
}

// nestingDedentVisitor removes one tab from every whitespace in a subtree,
// used to fix indentation when hoisting body statements out of an if block.
type nestingDedentVisitor struct {
	visitor.GoVisitor
}

func (v *nestingDedentVisitor) VisitSpace(space java.Space, p any) java.Space {
	if strings.Contains(space.Whitespace, "\t") {
		space.Whitespace = strings.Replace(space.Whitespace, "\t", "", 1)
	}
	return space
}
