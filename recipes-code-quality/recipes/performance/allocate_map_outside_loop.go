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

// AllocateMapOutsideLoop finds `make(map[...]...)` calls inside for/range loops
// and hoists the allocation before the loop, inserting `clear(m)` at the top of
// the loop body so the map is reused across iterations (Go 1.21+).
//
// Before:
//
//	for ... {
//	    m := make(map[K]V)
//	    m[k] = v
//	}
//
// After:
//
//	var m = make(map[K]V)
//	for ... {
//	    clear(m)
//	    m[k] = v
//	}
type AllocateMapOutsideLoop struct {
	recipe.Base
}

func (r *AllocateMapOutsideLoop) Name() string {
	return "org.openrewrite.golang.codequality.AllocateMapOutsideLoop"
}
func (r *AllocateMapOutsideLoop) DisplayName() string { return "Allocate map outside loop" }
func (r *AllocateMapOutsideLoop) Description() string {
	return "Hoist `make(map[...]...)` calls out of for/range loops and clear the map each iteration."
}
func (r *AllocateMapOutsideLoop) Tags() []string { return []string{"performance"} }

func (r *AllocateMapOutsideLoop) Editor() recipe.TreeVisitor {
	return visitor.Init(&allocateMapOutsideLoopVisitor{})
}

type allocateMapOutsideLoopVisitor struct {
	visitor.GoVisitor
}

// mapMakeInfo holds a found make(map[K]V) call inside a loop body.
type mapMakeInfo struct {
	stmtIdx  int                    // index in the loop body's statement list
	makeCall *tree.MethodInvocation // the make(...) call
	varName  string                 // the assigned variable name (e.g. "m")
}

func (v *allocateMapOutsideLoopVisitor) VisitBlock(block *tree.Block, p any) tree.J {
	block = v.GoVisitor.VisitBlock(block, p).(*tree.Block)

	var newStmts []tree.RightPadded[tree.Statement]
	changed := false

	for _, rp := range block.Statements {
		loopBody := getLoopBody(rp.Element)
		if loopBody == nil {
			newStmts = append(newStmts, rp)
			continue
		}

		found := findMapMakeCalls(loopBody)
		if len(found) == 0 {
			newStmts = append(newStmts, rp)
			continue
		}

		loopStmt := rp.Element
		for _, mi := range found {
			prefix := stmtPrefix(rp.Element)

			// Build: var m = make(map[K]V)
			hoisted := buildMapVarDecl(mi.varName, mi.makeCall, prefix)
			newStmts = append(newStmts, tree.RightPadded[tree.Statement]{Element: hoisted})

			// Replace the m := make(map[K]V) with clear(m) in the loop body.
			loopStmt = replaceMapMakeInLoop(loopStmt, mi)
		}
		newStmts = append(newStmts, tree.RightPadded[tree.Statement]{Element: loopStmt, After: rp.After})
		changed = true
	}

	if changed {
		return block.WithStatements(newStmts)
	}
	return block
}

// findMapMakeCalls scans a loop body for short-var-decl assignments of the form
// m := make(map[K]V) or m := make(map[K]V, size).
func findMapMakeCalls(body *tree.Block) []mapMakeInfo {
	var results []mapMakeInfo
	for i, rp := range body.Statements {
		assign, ok := rp.Element.(*tree.Assignment)
		if !ok {
			continue
		}
		// Must be a short variable declaration (:=).
		if tree.FindMarker[tree.ShortVarDecl](assign.Markers) == nil {
			continue
		}
		mi := extractMapMake(assign.Value.Element)
		if mi == nil {
			continue
		}
		ident, ok := assign.Variable.(*tree.Identifier)
		if !ok {
			continue
		}
		results = append(results, mapMakeInfo{
			stmtIdx:  i,
			makeCall: mi,
			varName:  ident.Name,
		})
	}
	return results
}

// extractMapMake checks if an expression is make(map[K]V, ...).
func extractMapMake(expr tree.Expression) *tree.MethodInvocation {
	mi, ok := expr.(*tree.MethodInvocation)
	if !ok || mi.Select != nil || mi.Name.Name != "make" {
		return nil
	}
	var realArgs []tree.Expression
	for _, arg := range mi.Arguments.Elements {
		if _, isEmpty := arg.Element.(*tree.Empty); !isEmpty {
			realArgs = append(realArgs, arg.Element)
		}
	}
	if len(realArgs) == 0 {
		return nil
	}
	if _, isMap := realArgs[0].(*tree.MapType); !isMap {
		return nil
	}
	return mi
}

// buildMapVarDecl constructs: var <name> = make(map[K]V)
func buildMapVarDecl(name string, call *tree.MethodInvocation, prefix tree.Space) *tree.VariableDeclarations {
	// Clone the call expression. The leading space between "=" and "make" is
	// handled by the LeftPadded.Before, so the call itself needs empty prefix.
	cleanCall := *call
	cleanCall.ID = uuid.New()

	nameIdent := &tree.Identifier{
		ID:   uuid.New(),
		Name: name,
	}
	init := &tree.LeftPadded[tree.Expression]{
		Before:  tree.Space{Whitespace: " "},
		Element: &cleanCall,
	}
	vd := &tree.VariableDeclarator{
		ID:          uuid.New(),
		Prefix:      tree.Space{Whitespace: " "},
		Name:        nameIdent,
		Initializer: init,
	}
	return &tree.VariableDeclarations{
		ID:     uuid.New(),
		Prefix: prefix,
		Markers: tree.Markers{
			ID:      uuid.New(),
			Entries: []tree.Marker{tree.VarKeyword{Ident: uuid.New()}},
		},
		Variables: []tree.RightPadded[*tree.VariableDeclarator]{
			{Element: vd},
		},
	}
}

// replaceMapMakeInLoop replaces the make(map[K]V) assignment at the given
// statement index with a clear(m) call.
func replaceMapMakeInLoop(loopStmt tree.Statement, mi mapMakeInfo) tree.Statement {
	switch loop := loopStmt.(type) {
	case *tree.ForLoop:
		newBody := replaceMapMakeInBody(loop.Body, mi)
		return loop.WithBody(newBody)
	case *tree.ForEachLoop:
		newBody := replaceMapMakeInBody(loop.Body, mi)
		return loop.WithBody(newBody)
	}
	return loopStmt
}

// replaceMapMakeInBody replaces the assignment at the given index with clear(m).
func replaceMapMakeInBody(body *tree.Block, mi mapMakeInfo) *tree.Block {
	newStmts := make([]tree.RightPadded[tree.Statement], len(body.Statements))
	copy(newStmts, body.Statements)

	rp := newStmts[mi.stmtIdx]

	// Determine the indentation prefix from the original statement.
	stmtWs := ""
	if assign, ok := rp.Element.(*tree.Assignment); ok {
		if ident, ok := assign.Variable.(*tree.Identifier); ok {
			stmtWs = ident.Prefix.Whitespace
		}
	}

	// Build: clear(m)
	clearCall := &tree.MethodInvocation{
		ID: uuid.New(),
		Name: &tree.Identifier{
			ID:     uuid.New(),
			Prefix: tree.Space{Whitespace: stmtWs},
			Name:   "clear",
		},
		Arguments: tree.Container[tree.Expression]{
			Before: tree.EmptySpace,
			Elements: []tree.RightPadded[tree.Expression]{
				{Element: &tree.Identifier{
					ID:   uuid.New(),
					Name: mi.varName,
				}},
			},
		},
	}

	newStmts[mi.stmtIdx] = tree.RightPadded[tree.Statement]{Element: clearCall, After: rp.After}
	return body.WithStatements(newStmts)
}
