/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package performance

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// UseStringsBuilderInLoop finds `s += expr` (compound string concatenation)
// inside for/range loops. Repeated string concatenation in loops is
// inefficient because strings are immutable in Go; use strings.Builder instead.
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
	return "Find `s += expr` inside for/range loops. Repeated string concatenation in loops is inefficient; consider using strings.Builder."
}
func (r *UseStringsBuilderInLoop) Tags() []string { return []string{"performance"} }

func (r *UseStringsBuilderInLoop) Editor() recipe.TreeVisitor {
	return visitor.Init(&useStringsBuilderInLoopVisitor{})
}

type useStringsBuilderInLoopVisitor struct {
	visitor.GoVisitor
	insideLoop int
}

func (v *useStringsBuilderInLoopVisitor) VisitForLoop(forLoop *tree.ForLoop, p any) tree.J {
	v.insideLoop++
	forLoop = v.GoVisitor.VisitForLoop(forLoop, p).(*tree.ForLoop)
	v.insideLoop--
	return forLoop
}

func (v *useStringsBuilderInLoopVisitor) VisitForEachLoop(forEach *tree.ForEachLoop, p any) tree.J {
	v.insideLoop++
	forEach = v.GoVisitor.VisitForEachLoop(forEach, p).(*tree.ForEachLoop)
	v.insideLoop--
	return forEach
}

func (v *useStringsBuilderInLoopVisitor) VisitAssignmentOperation(ao *tree.AssignmentOperation, p any) tree.J {
	ao = v.GoVisitor.VisitAssignmentOperation(ao, p).(*tree.AssignmentOperation)

	if v.insideLoop == 0 {
		return ao
	}

	// Only match += (AddAssign) operations.
	if ao.Operator.Element != tree.AddAssign {
		return ao
	}

	ao = ao.WithMarkers(
		tree.MarkupInfo(ao.Markers, "string concatenation in loop; consider strings.Builder"),
	)
	return ao
}
