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

// UseStringsBuilderInLoop finds `s += expr` (compound string concatenation)
// inside for/range loops and rewrites the code to use strings.Builder:
//
//	var builder strings.Builder
//	for ... { builder.WriteString(expr) }
//	s = builder.String()
type UseStringsBuilderInLoop struct {
	recipe.Base
}

func (r *UseStringsBuilderInLoop) Name() string {
	return "org.openrewrite.golang.codequality.UseStringsBuilderInLoop"
}

func (r *UseStringsBuilderInLoop) DisplayName() string {
	return "Use strings.Builder in loop"
}

func (r *UseStringsBuilderInLoop) Description() string {
	return "Find `s += expr` inside for/range loops. Repeated string concatenation in loops is inefficient; rewrite to use strings.Builder."
}

func (r *UseStringsBuilderInLoop) Tags() []string { return []string{"performance"} }

func (r *UseStringsBuilderInLoop) Editor() recipe.TreeVisitor {
	return visitor.Init(&useStringsBuilderInLoopVisitor{})
}

type useStringsBuilderInLoopVisitor struct {
	visitor.GoVisitor
	needsStringsImport bool
}

// stringConcatInfo records a string += found in a loop body.
type stringConcatInfo struct {
	stmtIdx  int             // index in loop body statement list
	variable java.Expression // the LHS variable (e.g. "s")
	rhs      java.Expression // the RHS expression (e.g. "item")
}

func (v *useStringsBuilderInLoopVisitor) VisitCompilationUnit(cu *golang.CompilationUnit, p any) java.J {
	v.needsStringsImport = false
	cu = v.GoVisitor.VisitCompilationUnit(cu, p).(*golang.CompilationUnit)

	if !v.needsStringsImport {
		return cu
	}

	// Check if "strings" is already imported.
	if cu.Imports != nil {
		for _, rp := range cu.Imports.Elements {
			if lit, ok := rp.Element.Qualid.(*java.Literal); ok {
				if lit.Source == `"strings"` {
					return cu
				}
			}
		}
	}

	if cu.Imports != nil {
		// Append to existing grouped imports.
		newImport := &java.Import{
			ID:     uuid.New(),
			Prefix: java.Space{Whitespace: "\n\t"},
			Qualid: &java.Literal{
				ID:     uuid.New(),
				Prefix: java.SingleSpace,
				Kind:   java.StringLiteral,
				Source: `"strings"`,
				Value:  "strings",
			},
		}
		imports := *cu.Imports
		imports.Elements = append(imports.Elements, java.RightPadded[*java.Import]{Element: newImport})
		cu = cu.WithImports(&imports)
	} else {
		// No imports exist yet: create a standalone import "strings".
		// Container.Before = space before the `import` keyword.
		// Import has no prefix; Qualid.Prefix = space between `import` and path.
		standaloneImport := &java.Import{
			ID: uuid.New(),
			Qualid: &java.Literal{
				ID:     uuid.New(),
				Prefix: java.SingleSpace,
				Kind:   java.StringLiteral,
				Source: `"strings"`,
				Value:  "strings",
			},
		}
		cu = cu.WithImports(&java.Container[*java.Import]{
			Before: java.Space{Whitespace: "\n\n"},
			Elements: []java.RightPadded[*java.Import]{
				{Element: standaloneImport},
			},
		})
	}

	return cu
}

func (v *useStringsBuilderInLoopVisitor) VisitBlock(block *java.Block, p any) java.J {
	block = v.GoVisitor.VisitBlock(block, p).(*java.Block)

	var newStmts []java.RightPadded[java.Statement]
	changed := false

	for _, rp := range block.Statements {
		loopBody := getLoopBody(rp.Element)
		if loopBody == nil {
			newStmts = append(newStmts, rp)
			continue
		}

		// Find string concatenation assignments (s += expr) inside the loop body.
		found := findStringConcats(loopBody)
		if len(found) == 0 {
			newStmts = append(newStmts, rp)
			continue
		}

		// We only handle the first concat per loop for simplicity.
		sc := found[0]
		prefix := stmtPrefix(rp.Element)

		// 1. Insert: var builder strings.Builder
		builderDecl := buildBuilderVarDecl(prefix)
		newStmts = append(newStmts, java.RightPadded[java.Statement]{Element: builderDecl})

		// 2. Replace s += expr with builder.WriteString(expr) inside the loop.
		modifiedLoop := replaceAddAssignInLoop(rp.Element, sc)
		newStmts = append(newStmts, java.RightPadded[java.Statement]{Element: modifiedLoop, After: rp.After})

		// 3. Insert: s = builder.String()
		assignStmt := buildBuilderStringAssign(sc.variable, prefix)
		newStmts = append(newStmts, java.RightPadded[java.Statement]{Element: assignStmt})

		changed = true
		v.needsStringsImport = true
	}

	if changed {
		return block.WithStatements(newStmts)
	}
	return block
}

// findStringConcats scans a loop body block for s += expr (AddAssign) operations.
func findStringConcats(body *java.Block) []stringConcatInfo {
	var results []stringConcatInfo
	for i, rp := range body.Statements {
		ao, ok := rp.Element.(*java.AssignmentOperation)
		if !ok {
			continue
		}
		if ao.Operator.Element != java.AddAssign {
			continue
		}
		results = append(results, stringConcatInfo{
			stmtIdx:  i,
			variable: ao.Variable,
			rhs:      ao.Assignment,
		})
	}
	return results
}

// buildBuilderVarDecl constructs: var builder strings.Builder
func buildBuilderVarDecl(prefix java.Space) *java.VariableDeclarations {
	typeExpr := &java.FieldAccess{
		ID:     uuid.New(),
		Prefix: java.SingleSpace,
		Target: &java.Identifier{
			ID:   uuid.New(),
			Name: "strings",
		},
		Name: java.LeftPadded[*java.Identifier]{
			Element: &java.Identifier{
				ID:   uuid.New(),
				Name: "Builder",
			},
		},
	}

	nameIdent := &java.Identifier{
		ID:   uuid.New(),
		Name: "builder",
	}

	declarator := &java.VariableDeclarator{
		ID:     uuid.New(),
		Prefix: java.SingleSpace,
		Name:   nameIdent,
	}

	return &java.VariableDeclarations{
		ID:       uuid.New(),
		Prefix:   prefix,
		Markers:  java.Markers{ID: uuid.New(), Entries: []java.Marker{golang.VarKeyword{Ident: uuid.New()}}},
		TypeExpr: typeExpr,
		Variables: []java.RightPadded[*java.VariableDeclarator]{
			{Element: declarator},
		},
	}
}

// buildBuilderStringAssign constructs: s = builder.String()
// prefix is the loop-level indentation (e.g. "\n\t").
func buildBuilderStringAssign(variable java.Expression, prefix java.Space) *java.Assignment {
	builderString := &java.MethodInvocation{
		ID: uuid.New(),
		Select: &java.RightPadded[java.Expression]{
			Element: &java.Identifier{
				ID:     uuid.New(),
				Prefix: java.SingleSpace,
				Name:   "builder",
			},
		},
		Name: &java.Identifier{
			ID:   uuid.New(),
			Name: "String",
		},
		Arguments: java.Container[java.Expression]{
			Before: java.EmptySpace,
		},
	}

	// For expression-based statements (Assignment), the leading whitespace
	// goes on the first sub-expression (the LHS variable).
	varClone := cloneIdentWithPrefix(variable, prefix)

	return &java.Assignment{
		ID:       uuid.New(),
		Variable: varClone,
		Value: java.LeftPadded[java.Expression]{
			Before:  java.SingleSpace,
			Element: builderString,
		},
	}
}

// replaceAddAssignInLoop replaces s += expr with builder.WriteString(expr) in the loop body.
func replaceAddAssignInLoop(loopStmt java.Statement, sc stringConcatInfo) java.Statement {
	switch loop := loopStmt.(type) {
	case *java.ForLoop:
		newBody := replaceAddAssignInBody(loop.Body, sc)
		return loop.WithBody(newBody)
	case *java.ForEachLoop:
		newBody := replaceAddAssignInBody(loop.Body, sc)
		return loop.WithBody(newBody)
	}
	return loopStmt
}

// replaceAddAssignInBody replaces the AssignmentOperation at the given index
// with a builder.WriteString(expr) call.
func replaceAddAssignInBody(body *java.Block, sc stringConcatInfo) *java.Block {
	newStmts := make([]java.RightPadded[java.Statement], len(body.Statements))
	copy(newStmts, body.Statements)

	rp := newStmts[sc.stmtIdx]
	ao := rp.Element.(*java.AssignmentOperation)

	// For expression-based statements, the leading whitespace is on
	// Variable.Prefix, not on the statement itself.
	varPrefix := java.EmptySpace
	if ident, ok := ao.Variable.(*java.Identifier); ok {
		varPrefix = ident.Prefix
	}

	// Build: builder.WriteString(expr)
	// Put the leading whitespace on the Select element (builder identifier).
	writeCall := &java.MethodInvocation{
		ID: uuid.New(),
		Select: &java.RightPadded[java.Expression]{
			Element: &java.Identifier{
				ID:     uuid.New(),
				Prefix: varPrefix,
				Name:   "builder",
			},
		},
		Name: &java.Identifier{
			ID:   uuid.New(),
			Name: "WriteString",
		},
		Arguments: java.Container[java.Expression]{
			Before: java.EmptySpace,
			Elements: []java.RightPadded[java.Expression]{
				{Element: setExprPrefix(sc.rhs, java.EmptySpace)},
			},
		},
	}

	newStmts[sc.stmtIdx] = java.RightPadded[java.Statement]{Element: writeCall, After: rp.After}
	return body.WithStatements(newStmts)
}

// cloneIdentWithPrefix creates a copy of an Identifier expression with a new prefix.
func cloneIdentWithPrefix(expr java.Expression, prefix java.Space) java.Expression {
	if ident, ok := expr.(*java.Identifier); ok {
		return &java.Identifier{
			ID:     uuid.New(),
			Prefix: prefix,
			Name:   ident.Name,
		}
	}
	return expr
}

// setExprPrefix sets the prefix on an expression.
func setExprPrefix(expr java.Expression, prefix java.Space) java.Expression {
	switch n := expr.(type) {
	case *java.Identifier:
		return n.WithPrefix(prefix)
	case *java.Literal:
		return n.WithPrefix(prefix)
	case *java.MethodInvocation:
		return n.WithPrefix(prefix)
	case *java.FieldAccess:
		return n.WithPrefix(prefix)
	case *java.MethodDeclaration:
		return n.WithPrefix(prefix)
	default:
		return expr
	}
}
