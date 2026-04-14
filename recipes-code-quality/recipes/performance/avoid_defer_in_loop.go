/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package performance

import (
	"github.com/google/uuid"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// AvoidDeferInLoop finds `defer` statements inside for/range loops and wraps
// the loop body in an immediately-invoked function literal so that defers
// run per-iteration instead of accumulating until function exit:
//
//	for ... { func() { defer x(); ...original body... }() }
type AvoidDeferInLoop struct {
	recipe.Base
}

func (r *AvoidDeferInLoop) Name() string {
	return "org.openrewrite.golang.codequality.AvoidDeferInLoop"
}

func (r *AvoidDeferInLoop) DisplayName() string { return "Avoid defer in loop" }

func (r *AvoidDeferInLoop) Description() string {
	return "Wrap loop bodies containing `defer` in an immediately-invoked function literal so deferred calls run per iteration."
}

func (r *AvoidDeferInLoop) Tags() []string { return []string{"performance"} }

func (r *AvoidDeferInLoop) Editor() recipe.TreeVisitor {
	return visitor.Init(&avoidDeferInLoopVisitor{})
}

type avoidDeferInLoopVisitor struct {
	visitor.GoVisitor
}

func (v *avoidDeferInLoopVisitor) VisitBlock(block *tree.Block, p any) tree.J {
	block = v.GoVisitor.VisitBlock(block, p).(*tree.Block)

	var newStmts []tree.RightPadded[tree.Statement]
	changed := false

	for _, rp := range block.Statements {
		loopBody := getLoopBody(rp.Element)
		if loopBody == nil {
			newStmts = append(newStmts, rp)
			continue
		}

		if !bodyContainsDefer(loopBody) {
			newStmts = append(newStmts, rp)
			continue
		}

		wrappedLoop := wrapLoopBodyInFunc(rp.Element, loopBody)
		newStmts = append(newStmts, tree.RightPadded[tree.Statement]{Element: wrappedLoop, After: rp.After})
		changed = true
	}

	if changed {
		return block.WithStatements(newStmts)
	}
	return block
}

// bodyContainsDefer checks whether a block contains any Defer statement
// at the top level.
func bodyContainsDefer(body *tree.Block) bool {
	for _, rp := range body.Statements {
		if _, ok := rp.Element.(*tree.Defer); ok {
			return true
		}
	}
	return false
}

// wrapLoopBodyInFunc replaces the loop body with a single IIFE statement.
func wrapLoopBodyInFunc(loopStmt tree.Statement, body *tree.Block) tree.Statement {
	wrapped := buildIIFEBlock(body)
	switch loop := loopStmt.(type) {
	case *tree.ForLoop:
		return loop.WithBody(wrapped)
	case *tree.ForEachLoop:
		return loop.WithBody(wrapped)
	}
	return loopStmt
}

// buildIIFEBlock wraps the original body in func() { ... }().
//
// It re-indents statements one level deeper inside the function literal.
func buildIIFEBlock(originalBody *tree.Block) *tree.Block {
	// Determine the current statement indentation from the first statement.
	stmtIndent := extractStmtIndent(originalBody)

	// Inner body statements need to be indented one level deeper.
	deeperIndent := stmtIndent + "\t"
	innerStmts := make([]tree.RightPadded[tree.Statement], len(originalBody.Statements))
	for i, rp := range originalBody.Statements {
		innerStmts[i] = tree.RightPadded[tree.Statement]{
			Element: setStmtPrefix(rp.Element, tree.Space{Whitespace: deeperIndent}),
			After:   rp.After,
		}
	}

	// Inner body End = same indent as the IIFE call.
	innerEnd := tree.Space{Whitespace: stmtIndent}

	// Build: func() { ...indented body... }
	// The leading whitespace goes on the MethodDeclaration prefix since
	// the printer emits md.Prefix then "func".
	funcLit := &tree.MethodDeclaration{
		ID:     uuid.New(),
		Prefix: tree.Space{Whitespace: stmtIndent},
		Name: &tree.Identifier{
			ID: uuid.New(),
		},
		Parameters: tree.Container[tree.Statement]{
			Before: tree.EmptySpace,
		},
		Body: &tree.Block{
			ID:         uuid.New(),
			Prefix:     tree.SingleSpace,
			Statements: innerStmts,
			End:        innerEnd,
		},
	}

	// Build: func() { ... }()
	// mi.Prefix is empty; the leading whitespace lives on funcLit.Prefix.
	iifeCall := &tree.MethodInvocation{
		ID: uuid.New(),
		Select: &tree.RightPadded[tree.Expression]{
			Element: funcLit,
		},
		Name: &tree.Identifier{
			ID: uuid.New(),
		},
		Arguments: tree.Container[tree.Expression]{
			Before: tree.EmptySpace,
		},
	}

	// Outer block preserves original prefix and End.
	return &tree.Block{
		ID:     uuid.New(),
		Prefix: originalBody.Prefix,
		Statements: []tree.RightPadded[tree.Statement]{
			{Element: iifeCall},
		},
		End: originalBody.End,
	}
}

// extractStmtIndent returns the indentation string from the first statement
// in a block, including the leading newline.
func extractStmtIndent(body *tree.Block) string {
	if len(body.Statements) == 0 {
		return "\n\t"
	}
	stmt := body.Statements[0].Element
	ws := getStmtWhitespace(stmt)
	if ws != "" {
		return ws
	}
	// Fallback: look at sub-expressions.
	return firstExprPrefix(stmt)
}

// getStmtWhitespace returns the Whitespace field of a statement's Prefix.
func getStmtWhitespace(stmt tree.Statement) string {
	switch s := stmt.(type) {
	case *tree.Defer:
		return s.Prefix.Whitespace
	case *tree.Return:
		return s.Prefix.Whitespace
	case *tree.ForLoop:
		return s.Prefix.Whitespace
	case *tree.ForEachLoop:
		return s.Prefix.Whitespace
	case *tree.If:
		return s.Prefix.Whitespace
	case *tree.VariableDeclarations:
		return s.Prefix.Whitespace
	default:
		return ""
	}
}

// firstExprPrefix extracts the whitespace from the first sub-expression of
// an expression-based statement (e.g. AssignmentOperation, MultiAssignment).
func firstExprPrefix(stmt tree.Statement) string {
	switch s := stmt.(type) {
	case *tree.AssignmentOperation:
		if ident, ok := s.Variable.(*tree.Identifier); ok {
			return ident.Prefix.Whitespace
		}
	case *tree.Assignment:
		if ident, ok := s.Variable.(*tree.Identifier); ok {
			return ident.Prefix.Whitespace
		}
	case *tree.MultiAssignment:
		if len(s.Variables) > 0 {
			if ident, ok := s.Variables[0].Element.(*tree.Identifier); ok {
				return ident.Prefix.Whitespace
			}
		}
	case *tree.MethodInvocation:
		if s.Select != nil {
			if ident, ok := s.Select.Element.(*tree.Identifier); ok {
				return ident.Prefix.Whitespace
			}
		}
		return s.Prefix.Whitespace
	}
	return "\n\t"
}

// setStmtPrefix sets the leading whitespace on a statement. For keyword-based
// statements (defer, return, var, for, if) the prefix is on the statement
// itself. For expression-based statements (assignments, method calls) the
// prefix is on the first sub-expression.
func setStmtPrefix(stmt tree.Statement, prefix tree.Space) tree.Statement {
	switch s := stmt.(type) {
	case *tree.Defer:
		return s.WithPrefix(prefix)
	case *tree.Return:
		return s.WithPrefix(prefix)
	case *tree.ForLoop:
		return s.WithPrefix(prefix)
	case *tree.ForEachLoop:
		return s.WithPrefix(prefix)
	case *tree.If:
		return s.WithPrefix(prefix)
	case *tree.VariableDeclarations:
		return s.WithPrefix(prefix)
	case *tree.AssignmentOperation:
		return s.WithVariable(setExprPrefix(s.Variable, prefix))
	case *tree.Assignment:
		c := *s
		c.Variable = setExprPrefix(s.Variable, prefix)
		return &c
	case *tree.MultiAssignment:
		if len(s.Variables) > 0 {
			c := *s
			vars := make([]tree.RightPadded[tree.Expression], len(s.Variables))
			copy(vars, s.Variables)
			vars[0] = tree.RightPadded[tree.Expression]{
				Element: setExprPrefix(s.Variables[0].Element, prefix),
				After:   s.Variables[0].After,
			}
			c.Variables = vars
			return &c
		}
		return stmt
	case *tree.MethodInvocation:
		if s.Select != nil {
			c := *s
			sel := *s.Select
			sel.Element = setExprPrefix(sel.Element, prefix)
			c.Select = &sel
			return &c
		}
		return s.WithPrefix(prefix)
	default:
		return stmt
	}
}

