/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package performance

import (
	"github.com/google/uuid"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
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

func (v *avoidDeferInLoopVisitor) VisitBlock(block *java.Block, p any) java.J {
	block = v.GoVisitor.VisitBlock(block, p).(*java.Block)

	var newStmts []java.RightPadded[java.Statement]
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
		newStmts = append(newStmts, java.RightPadded[java.Statement]{Element: wrappedLoop, After: rp.After})
		changed = true
	}

	if changed {
		return block.WithStatements(newStmts)
	}
	return block
}

// bodyContainsDefer checks whether a block contains any Defer statement
// at the top level.
func bodyContainsDefer(body *java.Block) bool {
	for _, rp := range body.Statements {
		if _, ok := rp.Element.(*golang.Defer); ok {
			return true
		}
	}
	return false
}

// wrapLoopBodyInFunc replaces the loop body with a single IIFE statement.
func wrapLoopBodyInFunc(loopStmt java.Statement, body *java.Block) java.Statement {
	wrapped := buildIIFEBlock(body)
	switch loop := loopStmt.(type) {
	case *java.ForLoop:
		return loop.WithBody(wrapped)
	case *java.ForEachLoop:
		return loop.WithBody(wrapped)
	}
	return loopStmt
}

// buildIIFEBlock wraps the original body in func() { ... }().
//
// It re-indents statements one level deeper inside the function literal.
func buildIIFEBlock(originalBody *java.Block) *java.Block {
	// Determine the current statement indentation from the first statement.
	stmtIndent := extractStmtIndent(originalBody)

	// Inner body statements need to be indented one level deeper.
	deeperIndent := stmtIndent + "\t"
	innerStmts := make([]java.RightPadded[java.Statement], len(originalBody.Statements))
	for i, rp := range originalBody.Statements {
		innerStmts[i] = java.RightPadded[java.Statement]{
			Element: setStmtPrefix(rp.Element, java.Space{Whitespace: deeperIndent}),
			After:   rp.After,
		}
	}

	// Inner body End = same indent as the IIFE call.
	innerEnd := java.Space{Whitespace: stmtIndent}

	// Build: func() { ...indented body... }
	// The leading whitespace goes on the MethodDeclaration prefix since
	// the printer emits md.Prefix then "func".
	funcLit := &java.MethodDeclaration{
		ID:     uuid.New(),
		Prefix: java.Space{Whitespace: stmtIndent},
		Name: &java.Identifier{
			ID: uuid.New(),
		},
		Parameters: java.Container[java.Statement]{
			Before: java.EmptySpace,
		},
		Body: &java.Block{
			ID:         uuid.New(),
			Prefix:     java.SingleSpace,
			Statements: innerStmts,
			End:        innerEnd,
		},
	}

	// Build: func() { ... }()
	// mi.Prefix is empty; the leading whitespace lives on funcLit.Prefix.
	iifeCall := &java.MethodInvocation{
		ID: uuid.New(),
		Select: &java.RightPadded[java.Expression]{
			Element: funcLit,
		},
		Name: &java.Identifier{
			ID: uuid.New(),
		},
		Arguments: java.Container[java.Expression]{
			Before: java.EmptySpace,
		},
	}

	// Outer block preserves original prefix and End.
	return &java.Block{
		ID:     uuid.New(),
		Prefix: originalBody.Prefix,
		Statements: []java.RightPadded[java.Statement]{
			{Element: iifeCall},
		},
		End: originalBody.End,
	}
}

// extractStmtIndent returns the indentation string from the first statement
// in a block, including the leading newline.
func extractStmtIndent(body *java.Block) string {
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
func getStmtWhitespace(stmt java.Statement) string {
	switch s := stmt.(type) {
	case *golang.Defer:
		return s.Prefix.Whitespace
	case *java.Return:
		return s.Prefix.Whitespace
	case *java.ForLoop:
		return s.Prefix.Whitespace
	case *java.ForEachLoop:
		return s.Prefix.Whitespace
	case *java.If:
		return s.Prefix.Whitespace
	case *java.VariableDeclarations:
		return s.Prefix.Whitespace
	default:
		return ""
	}
}

// firstExprPrefix extracts the whitespace from the first sub-expression of
// an expression-based statement (e.g. AssignmentOperation, MultiAssignment).
func firstExprPrefix(stmt java.Statement) string {
	switch s := stmt.(type) {
	case *java.AssignmentOperation:
		if ident, ok := s.Variable.(*java.Identifier); ok {
			return ident.Prefix.Whitespace
		}
	case *java.Assignment:
		if ident, ok := s.Variable.(*java.Identifier); ok {
			return ident.Prefix.Whitespace
		}
	case *golang.MultiAssignment:
		if len(s.Variables) > 0 {
			if ident, ok := s.Variables[0].Element.(*java.Identifier); ok {
				return ident.Prefix.Whitespace
			}
		}
	case *java.MethodInvocation:
		if s.Select != nil {
			if ident, ok := s.Select.Element.(*java.Identifier); ok {
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
func setStmtPrefix(stmt java.Statement, prefix java.Space) java.Statement {
	switch s := stmt.(type) {
	case *golang.Defer:
		return s.WithPrefix(prefix)
	case *java.Return:
		return s.WithPrefix(prefix)
	case *java.ForLoop:
		return s.WithPrefix(prefix)
	case *java.ForEachLoop:
		return s.WithPrefix(prefix)
	case *java.If:
		return s.WithPrefix(prefix)
	case *java.VariableDeclarations:
		return s.WithPrefix(prefix)
	case *java.AssignmentOperation:
		return s.WithVariable(setExprPrefix(s.Variable, prefix))
	case *java.Assignment:
		c := *s
		c.Variable = setExprPrefix(s.Variable, prefix)
		return &c
	case *golang.MultiAssignment:
		if len(s.Variables) > 0 {
			c := *s
			vars := make([]java.RightPadded[java.Expression], len(s.Variables))
			copy(vars, s.Variables)
			vars[0] = java.RightPadded[java.Expression]{
				Element: setExprPrefix(s.Variables[0].Element, prefix),
				After:   s.Variables[0].After,
			}
			c.Variables = vars
			return &c
		}
		return stmt
	case *java.MethodInvocation:
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
