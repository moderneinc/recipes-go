/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package performance

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// OptimizeCopyInLoop finds `copy()` calls inside for/range loops. Repeated
// copying in loops may indicate a buffer reuse opportunity.
type OptimizeCopyInLoop struct {
	recipe.Base
}

func (r *OptimizeCopyInLoop) Name() string {
	return "org.openrewrite.golang.codequality.OptimizeCopyInLoop"
}
func (r *OptimizeCopyInLoop) DisplayName() string { return "Optimize copy in loop" }
func (r *OptimizeCopyInLoop) Description() string {
	return "Find `copy()` calls inside for/range loops. Repeated copying in loops may indicate a buffer reuse opportunity."
}
func (r *OptimizeCopyInLoop) Tags() []string { return []string{"performance"} }

func (r *OptimizeCopyInLoop) Editor() recipe.TreeVisitor {
	return visitor.Init(&optimizeCopyInLoopVisitor{})
}

type optimizeCopyInLoopVisitor struct {
	visitor.GoVisitor
	insideLoop int
}

func (v *optimizeCopyInLoopVisitor) VisitForLoop(forLoop *tree.ForLoop, p any) tree.J {
	v.insideLoop++
	forLoop = v.GoVisitor.VisitForLoop(forLoop, p).(*tree.ForLoop)
	v.insideLoop--
	return forLoop
}

func (v *optimizeCopyInLoopVisitor) VisitForEachLoop(forEach *tree.ForEachLoop, p any) tree.J {
	v.insideLoop++
	forEach = v.GoVisitor.VisitForEachLoop(forEach, p).(*tree.ForEachLoop)
	v.insideLoop--
	return forEach
}

func (v *optimizeCopyInLoopVisitor) VisitMethodInvocation(mi *tree.MethodInvocation, p any) tree.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*tree.MethodInvocation)

	if v.insideLoop == 0 {
		return mi
	}

	// Match: copy(dst, src) — built-in, so no Select and Name == "copy".
	if mi.Select != nil || mi.Name.Name != "copy" {
		return mi
	}

	mi = mi.WithMarkers(
		tree.MarkupInfo(mi.Markers, "copy in loop; consider reusing buffer outside loop"),
	)
	return mi
}
