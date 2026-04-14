/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/parser"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// UseHttpServerWithTimeout replaces calls to `http.ListenAndServe(addr, handler)`
// with an explicit `http.Server` that has read and write timeouts configured,
// followed by a call to `server.ListenAndServe()`. The default http.Server has
// no timeouts, which makes the server vulnerable to denial-of-service attacks.
type UseHttpServerWithTimeout struct {
	recipe.Base
}

func (r *UseHttpServerWithTimeout) Name() string {
	return "org.openrewrite.golang.codequality.UseHttpServerWithTimeout"
}
func (r *UseHttpServerWithTimeout) DisplayName() string {
	return "Use http.Server with timeouts"
}
func (r *UseHttpServerWithTimeout) Description() string {
	return "Replace `http.ListenAndServe(addr, handler)` with an explicit `http.Server` with read/write timeouts."
}
func (r *UseHttpServerWithTimeout) Tags() []string { return []string{"security"} }

func (r *UseHttpServerWithTimeout) Editor() recipe.TreeVisitor {
	return visitor.Init(&useHttpServerWithTimeoutVisitor{})
}

type useHttpServerWithTimeoutVisitor struct {
	visitor.GoVisitor
}

func (v *useHttpServerWithTimeoutVisitor) VisitBlock(block *tree.Block, p any) tree.J {
	block = v.GoVisitor.VisitBlock(block, p).(*tree.Block)

	stmts := block.Statements
	var newStmts []tree.RightPadded[tree.Statement]
	changed := false

	for _, stmt := range stmts {
		mi, ok := stmt.Element.(*tree.MethodInvocation)
		if !ok {
			newStmts = append(newStmts, stmt)
			continue
		}

		if !isHttpListenAndServe(mi) {
			newStmts = append(newStmts, stmt)
			continue
		}

		// Extract the addr and handler arguments.
		args := nonEmptyArgs(mi.Arguments.Elements)
		if len(args) != 2 {
			newStmts = append(newStmts, stmt)
			continue
		}

		addrExpr := args[0]
		handlerExpr := args[1]

		// Build the two replacement statements via the parser.
		replacements := buildServerStatements(addrExpr, handlerExpr, mi)
		if replacements == nil {
			newStmts = append(newStmts, stmt)
			continue
		}

		for _, r := range replacements {
			newStmts = append(newStmts, tree.RightPadded[tree.Statement]{Element: r, After: stmt.After})
		}
		changed = true
	}

	if !changed {
		return block
	}
	return block.WithStatements(newStmts)
}

// isHttpListenAndServe checks if the method invocation is http.ListenAndServe.
func isHttpListenAndServe(mi *tree.MethodInvocation) bool {
	if mi.Select == nil {
		return false
	}
	ident, ok := mi.Select.Element.(*tree.Identifier)
	if !ok || ident.Name != "http" {
		return false
	}
	return mi.Name.Name == "ListenAndServe"
}

// nonEmptyArgs returns the non-empty arguments from an argument list.
func nonEmptyArgs(args []tree.RightPadded[tree.Expression]) []tree.Expression {
	var result []tree.Expression
	for _, a := range args {
		if _, isEmpty := a.Element.(*tree.Empty); !isEmpty {
			result = append(result, a.Element)
		}
	}
	return result
}

// buildServerStatements parses a scaffold containing the replacement statements
// and splices the captured addr/handler expressions into the result.
func buildServerStatements(addr, handler tree.Expression, original *tree.MethodInvocation) []tree.Statement {
	// Parse a scaffold that contains the two replacement statements with
	// placeholder identifiers that we will replace manually.
	source := `package __tmpl__

import (
	"net/http"
	"time"
)

func __f__() {
	server := &http.Server{Addr: __ADDR__, Handler: __HANDLER__, ReadTimeout: 10 * time.Second, WriteTimeout: 10 * time.Second}
	server.ListenAndServe()
}
`
	p := parser.NewGoParser()
	cu, err := p.Parse("__template__.go", source)
	if err != nil {
		return nil
	}

	// Find the function body.
	var bodyStmts []tree.RightPadded[tree.Statement]
	for _, stmt := range cu.Statements {
		md, ok := stmt.Element.(*tree.MethodDeclaration)
		if !ok || md.Name.Name != "__f__" || md.Body == nil {
			continue
		}
		bodyStmts = md.Body.Statements
		break
	}
	if len(bodyStmts) < 2 {
		return nil
	}

	// Extract the prefix from the original method invocation for formatting.
	prefix := httpMiPrefix(original)

	// First statement: server := &http.Server{Addr: __ADDR__, Handler: __HANDLER__, ...}
	declStmt := bodyStmts[0].Element
	declStmt = replaceIdentsInStatement(declStmt, map[string]tree.Expression{
		"__ADDR__":    addr,
		"__HANDLER__": handler,
	})
	declStmt = setStmtLeadingPrefix(declStmt, prefix)

	// Second statement: server.ListenAndServe()
	callStmt := bodyStmts[1].Element
	callStmt = setStmtLeadingPrefix(callStmt, prefix)

	return []tree.Statement{declStmt, callStmt}
}

// httpMiPrefix extracts the leading whitespace from an http.X method invocation.
func httpMiPrefix(mi *tree.MethodInvocation) tree.Space {
	if mi.Select != nil {
		if ident, ok := mi.Select.Element.(*tree.Identifier); ok {
			return ident.Prefix
		}
	}
	return mi.Name.Prefix
}

// replaceIdentsInStatement replaces named identifiers inside a statement AST node.
// This manually walks into Assignment values, Unary operands, and Composite elements
// to work around GoVisitor not recursing into Composite children.
func replaceIdentsInStatement(stmt tree.Statement, replacements map[string]tree.Expression) tree.Statement {
	switch s := stmt.(type) {
	case *tree.Assignment:
		val := replaceIdentsInExpression(s.Value.Element, replacements)
		return &tree.Assignment{
			ID: s.ID, Prefix: s.Prefix, Markers: s.Markers,
			Variable: s.Variable,
			Value:    tree.LeftPadded[tree.Expression]{Before: s.Value.Before, Element: val},
			Type:     s.Type,
		}
	default:
		return stmt
	}
}

// replaceIdentsInExpression recursively replaces named identifiers in an expression.
func replaceIdentsInExpression(expr tree.Expression, replacements map[string]tree.Expression) tree.Expression {
	switch e := expr.(type) {
	case *tree.Identifier:
		if replacement, ok := replacements[e.Name]; ok {
			return setExprPrefix(replacement, e.Prefix)
		}
		return e
	case *tree.Unary:
		replaced := replaceIdentsInExpression(e.Operand, replacements)
		return e.WithOperand(replaced)
	case *tree.Composite:
		newElements := make([]tree.RightPadded[tree.Expression], len(e.Elements.Elements))
		for i, elem := range e.Elements.Elements {
			newElements[i] = tree.RightPadded[tree.Expression]{
				Element: replaceIdentsInExpression(elem.Element, replacements),
				After:   elem.After,
				Markers: elem.Markers,
			}
		}
		return &tree.Composite{
			ID: e.ID, Prefix: e.Prefix, Markers: e.Markers,
			TypeExpr: e.TypeExpr,
			Elements: tree.Container[tree.Expression]{
				Before:   e.Elements.Before,
				Elements: newElements,
				Markers:  e.Elements.Markers,
			},
		}
	case *tree.KeyValue:
		val := replaceIdentsInExpression(e.Value.Element, replacements)
		return &tree.KeyValue{
			ID: e.ID, Prefix: e.Prefix, Markers: e.Markers,
			Key:   e.Key,
			Value: tree.LeftPadded[tree.Expression]{Before: e.Value.Before, Element: val},
		}
	default:
		return expr
	}
}

// setExprPrefix sets the leading prefix on an expression node.
func setExprPrefix(expr tree.Expression, prefix tree.Space) tree.Expression {
	switch e := expr.(type) {
	case *tree.Identifier:
		return e.WithPrefix(prefix)
	case *tree.Literal:
		return e.WithPrefix(prefix)
	default:
		return expr
	}
}

// setStmtLeadingPrefix sets the leading prefix on a statement node.
func setStmtLeadingPrefix(stmt tree.Statement, prefix tree.Space) tree.Statement {
	switch s := stmt.(type) {
	case *tree.Assignment:
		return s.WithVariable(setExprPrefix(s.Variable, prefix).(tree.Expression))
	case *tree.MethodInvocation:
		if s.Select != nil {
			sel := *s.Select
			if ident, ok := sel.Element.(*tree.Identifier); ok {
				sel.Element = ident.WithPrefix(prefix)
				return &tree.MethodInvocation{
					ID: s.ID, Prefix: s.Prefix, Markers: s.Markers,
					Select: &sel, Name: s.Name, Arguments: s.Arguments, MethodType: s.MethodType,
				}
			}
		}
		return s.WithName(s.Name.WithPrefix(prefix))
	default:
		return stmt
	}
}
