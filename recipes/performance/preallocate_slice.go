/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package performance

import (
	"github.com/google/uuid"
	"github.com/moderneinc/recipes-go/diagnostic"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// PreallocateSlice gives a capacity to a slice that is made empty and then
// filled by appending over a range, as in `out := make([]int, 0)` followed by a
// loop appending to out.
// golangci-lint: prealloc
type PreallocateSlice struct {
	recipe.Base
}

func (r *PreallocateSlice) Name() string {
	return "org.openrewrite.golang.codequality.PreallocateSlice"
}
func (r *PreallocateSlice) DisplayName() string { return "Preallocate slice" }
func (r *PreallocateSlice) Description() string {
	return "Add a capacity to a slice made empty and then filled by appending over a range, so `out := make([]int, 0)` before `for _, x := range xs` becomes `make([]int, 0, len(xs))`. The capacity is a hint, so only the allocation changes."
}
func (r *PreallocateSlice) Tags() []string { return []string{"performance", "lint"} }

func (r *PreallocateSlice) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{
		{DiagnosticID: "prealloc", Tool: diagnostic.GolangciLint, HasFix: false},
	}
}

func (r *PreallocateSlice) Editor() recipe.TreeVisitor {
	return visitor.Init(&preallocateSliceVisitor{})
}

type preallocateSliceVisitor struct {
	visitor.GoVisitor
}

func (v *preallocateSliceVisitor) VisitBlock(block *java.Block, p any) java.J {
	block = v.GoVisitor.VisitBlock(block, p).(*java.Block)

	stmts := block.Statements
	newStmts := make([]java.RightPadded[java.Statement], len(stmts))
	copy(newStmts, stmts)
	changed := false

	for i := 0; i+1 < len(stmts); i++ {
		name, mk := emptySliceMake(stmts[i].Element)
		if mk == nil {
			continue
		}
		// Taking the loop that follows directly is what makes the iterable
		// known to be in scope where the capacity is inserted.
		forEach, ok := stmts[i+1].Element.(*java.ForEachLoop)
		if !ok {
			continue
		}
		// A call would be evaluated a second time by len().
		iterable, ok := forEach.Control.Iterable.Element.(*java.Identifier)
		if !ok {
			continue
		}
		if !appendsTo(forEach.Body, name, p) {
			continue
		}
		newStmts[i] = java.RightPadded[java.Statement]{
			Element: withCapacity(stmts[i].Element, mk, iterable.Name),
			After:   stmts[i].After,
			Markers: stmts[i].Markers,
		}
		changed = true
	}

	if !changed {
		return block
	}
	return block.WithStatements(newStmts)
}

// emptySliceMake matches `name := make([]T, 0)` and returns the name it binds
// along with the make call, or a nil call for anything else.
func emptySliceMake(stmt java.Statement) (string, *java.MethodInvocation) {
	assign, ok := stmt.(*java.Assignment)
	if !ok {
		return "", nil
	}
	target, ok := assign.Variable.(*java.Identifier)
	if !ok {
		return "", nil
	}
	mk, ok := assign.Value.Element.(*java.MethodInvocation)
	if !ok || mk.Select != nil || mk.Name.Name != "make" {
		return "", nil
	}
	args := mk.Arguments.Elements
	if len(args) != 2 {
		return "", nil
	}
	length, ok := args[1].Element.(*java.Literal)
	if !ok || length.Source != "0" {
		return "", nil
	}
	return target.Name, mk
}

// withCapacity returns stmt with `len(iterable)` added to its make call.
func withCapacity(stmt java.Statement, mk *java.MethodInvocation, iterable string) java.Statement {
	lenCall := &java.MethodInvocation{
		ID:     uuid.New(),
		Prefix: java.SingleSpace,
		Name:   &java.Identifier{ID: uuid.New(), Name: "len"},
		Arguments: java.Container[java.Expression]{
			Elements: []java.RightPadded[java.Expression]{
				{Element: &java.Identifier{ID: uuid.New(), Name: iterable}},
			},
		},
	}

	newMake := *mk
	newMake.Arguments = java.Container[java.Expression]{
		Before: mk.Arguments.Before,
		Elements: append(append([]java.RightPadded[java.Expression]{}, mk.Arguments.Elements...),
			java.RightPadded[java.Expression]{Element: lenCall}),
		Markers: mk.Arguments.Markers,
	}

	assign := *stmt.(*java.Assignment)
	assign.Value = java.LeftPadded[java.Expression]{
		Before:  assign.Value.Before,
		Element: &newMake,
		Markers: assign.Value.Markers,
	}
	return &assign
}

// appendsTo reports whether body appends to a slice named target, at any depth.
func appendsTo(body *java.Block, target string, p any) bool {
	finder := &appendFinder{target: target}
	visitor.Init(finder).Visit(body, p)
	return finder.found
}

type appendFinder struct {
	visitor.GoVisitor
	target string
	found  bool
}

func (v *appendFinder) VisitAssignment(assign *java.Assignment, p any) java.J {
	assign = v.GoVisitor.VisitAssignment(assign, p).(*java.Assignment)

	target, ok := assign.Variable.(*java.Identifier)
	if !ok || target.Name != v.target {
		return assign
	}
	if mi, ok := assign.Value.Element.(*java.MethodInvocation); ok &&
		mi.Select == nil && mi.Name.Name == "append" {
		v.found = true
	}
	return assign
}
