/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package performance

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindStringConcatInLoop finds `s += expr` (compound string concatenation)
// inside for/range loops. Repeated string concatenation in loops is
// inefficient because strings are immutable in Go; use strings.Builder instead.
type FindStringConcatInLoop struct {
	recipe.Base
}

func (r *FindStringConcatInLoop) Name() string {
	return "org.openrewrite.golang.codequality.FindStringConcatInLoop"
}
func (r *FindStringConcatInLoop) DisplayName() string {
	return "Find string concatenation in loop"
}
func (r *FindStringConcatInLoop) Description() string {
	return "Find `s += expr` inside for/range loops. Repeated string concatenation in loops is inefficient; consider using strings.Builder."
}
func (r *FindStringConcatInLoop) Tags() []string { return []string{"performance"} }

func (r *FindStringConcatInLoop) Editor() recipe.TreeVisitor {
	return visitor.Init(&findStringConcatInLoopVisitor{})
}

type findStringConcatInLoopVisitor struct {
	visitor.GoVisitor
	insideLoop int
}

func (v *findStringConcatInLoopVisitor) VisitForLoop(forLoop *tree.ForLoop, p any) tree.J {
	v.insideLoop++
	forLoop = v.GoVisitor.VisitForLoop(forLoop, p).(*tree.ForLoop)
	v.insideLoop--
	return forLoop
}

func (v *findStringConcatInLoopVisitor) VisitForEachLoop(forEach *tree.ForEachLoop, p any) tree.J {
	v.insideLoop++
	forEach = v.GoVisitor.VisitForEachLoop(forEach, p).(*tree.ForEachLoop)
	v.insideLoop--
	return forEach
}

func (v *findStringConcatInLoopVisitor) VisitAssignmentOperation(ao *tree.AssignmentOperation, p any) tree.J {
	ao = v.GoVisitor.VisitAssignmentOperation(ao, p).(*tree.AssignmentOperation)

	if v.insideLoop == 0 {
		return ao
	}

	// Only match += (AddAssign) operations.
	if ao.Operator.Element != tree.AddAssign {
		return ao
	}

	ao = ao.WithMarkers(
		tree.FoundSearchResult(ao.Markers, "string concatenation in loop; consider strings.Builder"),
	)
	return ao
}
