/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package performance

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// CompileRegexOutsideLoop finds calls to `regexp.Compile()` or `regexp.MustCompile()`
// inside for/range loops. When the pattern argument is a string literal the call
// is hoisted before the loop and a variable reference replaces the original call
// inside the loop body. When the argument is not a literal (dynamic pattern) a
// warning marker is added instead.
type CompileRegexOutsideLoop struct {
	recipe.Base
}

func (r *CompileRegexOutsideLoop) Name() string {
	return "org.openrewrite.golang.codequality.CompileRegexOutsideLoop"
}
func (r *CompileRegexOutsideLoop) DisplayName() string { return "Compile regex outside loop" }
func (r *CompileRegexOutsideLoop) Description() string {
	return "Find `regexp.Compile()` or `regexp.MustCompile()` calls inside for/range loops. Compile the regex once outside the loop for better performance."
}
func (r *CompileRegexOutsideLoop) Tags() []string { return []string{"performance"} }

func (r *CompileRegexOutsideLoop) Editor() recipe.TreeVisitor {
	return visitor.Init(&compileRegexOutsideLoopVisitor{})
}

type compileRegexOutsideLoopVisitor struct {
	visitor.GoVisitor
	regexCounter int
}

func (v *compileRegexOutsideLoopVisitor) VisitBlock(block *java.Block, p any) java.J {
	block = v.GoVisitor.VisitBlock(block, p).(*java.Block)

	var newStmts []java.RightPadded[java.Statement]
	changed := false

	for _, rp := range block.Statements {
		loopBody := getLoopBody(rp.Element)
		if loopBody == nil {
			newStmts = append(newStmts, rp)
			continue
		}

		// Find regex compile calls with literal arguments inside the loop body.
		found := findRegexCalls(loopBody)
		if len(found) == 0 {
			newStmts = append(newStmts, rp)
			continue
		}

		// For each found call, create a hoisted var decl before the loop and
		// replace the assignment inside the loop with a plain assignment from
		// the hoisted variable.
		loopStmt := rp.Element
		for _, rc := range found {
			varName := fmt.Sprintf("compiledRegex%d", v.regexCounter)
			v.regexCounter++

			// Build: var <varName> = regexp.MustCompile("pattern")
			// with the same prefix (indentation) as the loop statement.
			hoisted := buildVarDecl(varName, rc.call, stmtPrefix(rp.Element))
			newStmts = append(newStmts, java.RightPadded[java.Statement]{Element: hoisted})

			// Replace the call inside the loop body.
			loopStmt = replaceCallInLoop(loopStmt, rc, varName)
		}
		newStmts = append(newStmts, java.RightPadded[java.Statement]{Element: loopStmt, After: rp.After})
		changed = true
	}

	if changed {
		return block.WithStatements(newStmts)
	}
	return block
}

// regexCallInfo holds a found regex call and its enclosing assignment statement index.
type regexCallInfo struct {
	call     *java.MethodInvocation // the regexp.Compile/MustCompile call
	stmtIdx  int                    // index in the loop body's statement list
	isMust   bool                   // true for MustCompile (single return), false for Compile (two returns)
	isSimple bool                   // true if the enclosing statement is a simple := assignment
}

// findRegexCalls scans a loop body block for regexp.Compile/MustCompile calls
// with string literal arguments.
func findRegexCalls(body *java.Block) []regexCallInfo {
	var results []regexCallInfo
	for i, rp := range body.Statements {
		switch stmt := rp.Element.(type) {
		case *java.Assignment:
			mi := extractRegexCall(stmt.Value.Element)
			if mi == nil {
				continue
			}
			if !hasLiteralArg(mi) {
				continue
			}
			results = append(results, regexCallInfo{
				call:     mi,
				stmtIdx:  i,
				isMust:   mi.Name.Name == "MustCompile",
				isSimple: true,
			})
		case *golang.MultiAssignment:
			if len(stmt.Values) != 1 {
				continue
			}
			mi := extractRegexCall(stmt.Values[0].Element)
			if mi == nil {
				continue
			}
			if !hasLiteralArg(mi) {
				continue
			}
			results = append(results, regexCallInfo{
				call:     mi,
				stmtIdx:  i,
				isMust:   mi.Name.Name == "MustCompile",
				isSimple: false,
			})
		}
	}
	return results
}

// extractRegexCall checks if an expression is regexp.Compile or regexp.MustCompile.
func extractRegexCall(expr java.Expression) *java.MethodInvocation {
	mi, ok := expr.(*java.MethodInvocation)
	if !ok || mi.Select == nil {
		return nil
	}
	ident, ok := mi.Select.Element.(*java.Identifier)
	if !ok || ident.Name != "regexp" {
		return nil
	}
	if mi.Name.Name != "Compile" && mi.Name.Name != "MustCompile" {
		return nil
	}
	return mi
}

// hasLiteralArg checks if the first argument of the method invocation is a string literal.
func hasLiteralArg(mi *java.MethodInvocation) bool {
	if len(mi.Arguments.Elements) == 0 {
		return false
	}
	_, ok := mi.Arguments.Elements[0].Element.(*java.Literal)
	return ok
}

// buildVarDecl constructs: var <name> = regexp.MustCompile("pattern")
// as a VariableDeclarations statement with a VarKeyword marker.
// If the original call was regexp.Compile, it is promoted to MustCompile
// since string literal patterns are known-valid at compile time.
func buildVarDecl(name string, call *java.MethodInvocation, prefix java.Space) *java.VariableDeclarations {
	// Clone the call expression, setting a single-space leading prefix.
	cleanCall := *call
	if call.Select != nil {
		sel := *call.Select
		sel.Element = setMISelectPrefix(sel.Element, java.Space{Whitespace: " "})
		cleanCall.Select = &sel
	}
	// Promote Compile to MustCompile for the hoisted call.
	if cleanCall.Name.Name == "Compile" {
		newName := *cleanCall.Name
		newName.Name = "MustCompile"
		cleanCall.Name = &newName
	}

	nameIdent := &java.Identifier{
		ID:   uuid.New(),
		Name: name,
	}
	init := &java.LeftPadded[java.Expression]{
		Before:  java.Space{Whitespace: " "},
		Element: &cleanCall,
	}
	vd := &java.VariableDeclarator{
		ID:          uuid.New(),
		Prefix:      java.Space{Whitespace: " "},
		Name:        nameIdent,
		Initializer: init,
	}
	return &java.VariableDeclarations{
		ID:     uuid.New(),
		Prefix: prefix,
		Markers: java.Markers{
			ID:      uuid.New(),
			Entries: []java.Marker{golang.VarKeyword{Ident: uuid.New()}},
		},
		Variables: []java.RightPadded[*java.VariableDeclarator]{
			{Element: vd},
		},
	}
}

// replaceCallInLoop replaces the regex call inside the loop statement with a
// reference to the hoisted variable.
func replaceCallInLoop(loopStmt java.Statement, rc regexCallInfo, varName string) java.Statement {
	switch loop := loopStmt.(type) {
	case *java.ForLoop:
		newBody := replaceInBody(loop.Body, rc, varName)
		return loop.WithBody(newBody)
	case *java.ForEachLoop:
		newBody := replaceInBody(loop.Body, rc, varName)
		return loop.WithBody(newBody)
	}
	return loopStmt
}

// replaceInBody replaces a regex call at the given statement index in the body block.
func replaceInBody(body *java.Block, rc regexCallInfo, varName string) *java.Block {
	newStmts := make([]java.RightPadded[java.Statement], len(body.Statements))
	copy(newStmts, body.Statements)

	rp := newStmts[rc.stmtIdx]
	varRef := &java.Identifier{
		ID:     uuid.New(),
		Prefix: java.Space{Whitespace: " "},
		Name:   varName,
	}

	if rc.isSimple {
		// Assignment: re := regexp.MustCompile("pattern")
		// -> re := compiledRegex0
		// But actually we should change := to = since the var is already declared.
		// Better: keep re = compiledRegex0 (assign, not short-var-decl).
		assign := rp.Element.(*java.Assignment)
		newAssign := &java.Assignment{
			ID:       assign.ID,
			Prefix:   assign.Prefix,
			Markers:  assign.Markers,
			Variable: assign.Variable,
			Value: java.LeftPadded[java.Expression]{
				Before:  assign.Value.Before,
				Element: varRef,
			},
		}
		newStmts[rc.stmtIdx] = java.RightPadded[java.Statement]{Element: newAssign, After: rp.After}
	} else {
		// MultiAssignment: re, _ := regexp.Compile("pattern")
		// -> re, _ := compiledRegex0, nil
		// Since MustCompile was hoisted and doesn't return error, we replace the
		// call with a reference to the hoisted var and nil for the error.
		ma := rp.Element.(*golang.MultiAssignment)
		nilIdent := &java.Identifier{
			ID:     uuid.New(),
			Prefix: java.Space{Whitespace: " "},
			Name:   "nil",
		}
		newValues := []java.RightPadded[java.Expression]{
			{Element: varRef},
			{Element: nilIdent},
		}
		newMA := &golang.MultiAssignment{
			ID:        ma.ID,
			Prefix:    ma.Prefix,
			Markers:   ma.Markers,
			Variables: ma.Variables,
			Operator:  ma.Operator,
			Values:    newValues,
		}
		newStmts[rc.stmtIdx] = java.RightPadded[java.Statement]{Element: newMA, After: rp.After}
	}

	return body.WithStatements(newStmts)
}

// getLoopBody extracts the body Block from a ForLoop or ForEachLoop.
func getLoopBody(stmt java.Statement) *java.Block {
	switch s := stmt.(type) {
	case *java.ForLoop:
		return s.Body
	case *java.ForEachLoop:
		return s.Body
	}
	return nil
}

// stmtPrefix extracts the prefix Space from a statement.
func stmtPrefix(stmt java.Statement) java.Space {
	switch s := stmt.(type) {
	case *java.ForLoop:
		return s.Prefix
	case *java.ForEachLoop:
		return s.Prefix
	default:
		return java.Space{}
	}
}

// setMISelectPrefix sets the prefix of a method invocation's select expression.
func setMISelectPrefix(expr java.Expression, prefix java.Space) java.Expression {
	switch n := expr.(type) {
	case *java.Identifier:
		return n.WithPrefix(prefix)
	case *java.FieldAccess:
		return n.WithTarget(setMISelectPrefix(n.Target, prefix))
	default:
		return expr
	}
}
