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
	variable tree.Expression // the LHS variable (e.g. "s")
	rhs      tree.Expression // the RHS expression (e.g. "item")
}

func (v *useStringsBuilderInLoopVisitor) VisitCompilationUnit(cu *tree.CompilationUnit, p any) tree.J {
	v.needsStringsImport = false
	cu = v.GoVisitor.VisitCompilationUnit(cu, p).(*tree.CompilationUnit)

	if !v.needsStringsImport {
		return cu
	}

	// Check if "strings" is already imported.
	if cu.Imports != nil {
		for _, rp := range cu.Imports.Elements {
			if lit, ok := rp.Element.Qualid.(*tree.Literal); ok {
				if lit.Source == `"strings"` {
					return cu
				}
			}
		}
	}

	if cu.Imports != nil {
		// Append to existing grouped imports.
		newImport := &tree.Import{
			ID:     uuid.New(),
			Prefix: tree.Space{Whitespace: "\n\t"},
			Qualid: &tree.Literal{
				ID:     uuid.New(),
				Prefix: tree.SingleSpace,
				Kind:   tree.StringLiteral,
				Source: `"strings"`,
				Value:  "strings",
			},
		}
		imports := *cu.Imports
		imports.Elements = append(imports.Elements, tree.RightPadded[*tree.Import]{Element: newImport})
		cu = cu.WithImports(&imports)
	} else {
		// No imports exist yet: create a standalone import "strings".
		// Container.Before = space before the `import` keyword.
		// Import has no prefix; Qualid.Prefix = space between `import` and path.
		standaloneImport := &tree.Import{
			ID: uuid.New(),
			Qualid: &tree.Literal{
				ID:     uuid.New(),
				Prefix: tree.SingleSpace,
				Kind:   tree.StringLiteral,
				Source: `"strings"`,
				Value:  "strings",
			},
		}
		cu = cu.WithImports(&tree.Container[*tree.Import]{
			Before: tree.Space{Whitespace: "\n\n"},
			Elements: []tree.RightPadded[*tree.Import]{
				{Element: standaloneImport},
			},
		})
	}

	return cu
}

func (v *useStringsBuilderInLoopVisitor) VisitBlock(block *tree.Block, p any) tree.J {
	block = v.GoVisitor.VisitBlock(block, p).(*tree.Block)

	var newStmts []tree.RightPadded[tree.Statement]
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
		newStmts = append(newStmts, tree.RightPadded[tree.Statement]{Element: builderDecl})

		// 2. Replace s += expr with builder.WriteString(expr) inside the loop.
		modifiedLoop := replaceAddAssignInLoop(rp.Element, sc)
		newStmts = append(newStmts, tree.RightPadded[tree.Statement]{Element: modifiedLoop, After: rp.After})

		// 3. Insert: s = builder.String()
		assignStmt := buildBuilderStringAssign(sc.variable, prefix)
		newStmts = append(newStmts, tree.RightPadded[tree.Statement]{Element: assignStmt})

		changed = true
		v.needsStringsImport = true
	}

	if changed {
		return block.WithStatements(newStmts)
	}
	return block
}

// findStringConcats scans a loop body block for s += expr (AddAssign) operations.
func findStringConcats(body *tree.Block) []stringConcatInfo {
	var results []stringConcatInfo
	for i, rp := range body.Statements {
		ao, ok := rp.Element.(*tree.AssignmentOperation)
		if !ok {
			continue
		}
		if ao.Operator.Element != tree.AddAssign {
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
func buildBuilderVarDecl(prefix tree.Space) *tree.VariableDeclarations {
	typeExpr := &tree.FieldAccess{
		ID:     uuid.New(),
		Prefix: tree.SingleSpace,
		Target: &tree.Identifier{
			ID:   uuid.New(),
			Name: "strings",
		},
		Name: tree.LeftPadded[*tree.Identifier]{
			Element: &tree.Identifier{
				ID:   uuid.New(),
				Name: "Builder",
			},
		},
	}

	nameIdent := &tree.Identifier{
		ID:   uuid.New(),
		Name: "builder",
	}

	declarator := &tree.VariableDeclarator{
		ID:     uuid.New(),
		Prefix: tree.SingleSpace,
		Name:   nameIdent,
	}

	return &tree.VariableDeclarations{
		ID:      uuid.New(),
		Prefix:  prefix,
		Markers: tree.Markers{ID: uuid.New(), Entries: []tree.Marker{tree.VarKeyword{Ident: uuid.New()}}},
		TypeExpr: typeExpr,
		Variables: []tree.RightPadded[*tree.VariableDeclarator]{
			{Element: declarator},
		},
	}
}

// buildBuilderStringAssign constructs: s = builder.String()
// prefix is the loop-level indentation (e.g. "\n\t").
func buildBuilderStringAssign(variable tree.Expression, prefix tree.Space) *tree.Assignment {
	builderString := &tree.MethodInvocation{
		ID: uuid.New(),
		Select: &tree.RightPadded[tree.Expression]{
			Element: &tree.Identifier{
				ID:     uuid.New(),
				Prefix: tree.SingleSpace,
				Name:   "builder",
			},
		},
		Name: &tree.Identifier{
			ID:   uuid.New(),
			Name: "String",
		},
		Arguments: tree.Container[tree.Expression]{
			Before: tree.EmptySpace,
		},
	}

	// For expression-based statements (Assignment), the leading whitespace
	// goes on the first sub-expression (the LHS variable).
	varClone := cloneIdentWithPrefix(variable, prefix)

	return &tree.Assignment{
		ID:       uuid.New(),
		Variable: varClone,
		Value: tree.LeftPadded[tree.Expression]{
			Before:  tree.SingleSpace,
			Element: builderString,
		},
	}
}

// replaceAddAssignInLoop replaces s += expr with builder.WriteString(expr) in the loop body.
func replaceAddAssignInLoop(loopStmt tree.Statement, sc stringConcatInfo) tree.Statement {
	switch loop := loopStmt.(type) {
	case *tree.ForLoop:
		newBody := replaceAddAssignInBody(loop.Body, sc)
		return loop.WithBody(newBody)
	case *tree.ForEachLoop:
		newBody := replaceAddAssignInBody(loop.Body, sc)
		return loop.WithBody(newBody)
	}
	return loopStmt
}

// replaceAddAssignInBody replaces the AssignmentOperation at the given index
// with a builder.WriteString(expr) call.
func replaceAddAssignInBody(body *tree.Block, sc stringConcatInfo) *tree.Block {
	newStmts := make([]tree.RightPadded[tree.Statement], len(body.Statements))
	copy(newStmts, body.Statements)

	rp := newStmts[sc.stmtIdx]
	ao := rp.Element.(*tree.AssignmentOperation)

	// For expression-based statements, the leading whitespace is on
	// Variable.Prefix, not on the statement itself.
	varPrefix := tree.EmptySpace
	if ident, ok := ao.Variable.(*tree.Identifier); ok {
		varPrefix = ident.Prefix
	}

	// Build: builder.WriteString(expr)
	// Put the leading whitespace on the Select element (builder identifier).
	writeCall := &tree.MethodInvocation{
		ID: uuid.New(),
		Select: &tree.RightPadded[tree.Expression]{
			Element: &tree.Identifier{
				ID:     uuid.New(),
				Prefix: varPrefix,
				Name:   "builder",
			},
		},
		Name: &tree.Identifier{
			ID:   uuid.New(),
			Name: "WriteString",
		},
		Arguments: tree.Container[tree.Expression]{
			Before: tree.EmptySpace,
			Elements: []tree.RightPadded[tree.Expression]{
				{Element: setExprPrefix(sc.rhs, tree.EmptySpace)},
			},
		},
	}

	newStmts[sc.stmtIdx] = tree.RightPadded[tree.Statement]{Element: writeCall, After: rp.After}
	return body.WithStatements(newStmts)
}

// cloneIdentWithPrefix creates a copy of an Identifier expression with a new prefix.
func cloneIdentWithPrefix(expr tree.Expression, prefix tree.Space) tree.Expression {
	if ident, ok := expr.(*tree.Identifier); ok {
		return &tree.Identifier{
			ID:     uuid.New(),
			Prefix: prefix,
			Name:   ident.Name,
		}
	}
	return expr
}

// setExprPrefix sets the prefix on an expression.
func setExprPrefix(expr tree.Expression, prefix tree.Space) tree.Expression {
	switch n := expr.(type) {
	case *tree.Identifier:
		return n.WithPrefix(prefix)
	case *tree.Literal:
		return n.WithPrefix(prefix)
	case *tree.MethodInvocation:
		return n.WithPrefix(prefix)
	case *tree.FieldAccess:
		return n.WithPrefix(prefix)
	case *tree.MethodDeclaration:
		return n.WithPrefix(prefix)
	default:
		return expr
	}
}
