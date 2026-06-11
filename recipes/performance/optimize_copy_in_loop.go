/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package performance

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
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

func (v *optimizeCopyInLoopVisitor) VisitForLoop(forLoop *java.ForLoop, p any) java.J {
	v.insideLoop++
	forLoop = v.GoVisitor.VisitForLoop(forLoop, p).(*java.ForLoop)
	v.insideLoop--
	return forLoop
}

func (v *optimizeCopyInLoopVisitor) VisitForEachLoop(forEach *java.ForEachLoop, p any) java.J {
	v.insideLoop++
	forEach = v.GoVisitor.VisitForEachLoop(forEach, p).(*java.ForEachLoop)
	v.insideLoop--
	return forEach
}

func (v *optimizeCopyInLoopVisitor) VisitMethodInvocation(mi *java.MethodInvocation, p any) java.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*java.MethodInvocation)

	if v.insideLoop == 0 {
		return mi
	}

	// Match: copy(dst, src) — built-in, so no Select and Name == "copy".
	if mi.Select != nil || mi.Name.Name != "copy" {
		return mi
	}

	mi = mi.WithMarkers(
		java.MarkupInfo(mi.Markers, "copy in loop; consider reusing buffer outside loop"),
	)
	return mi
}
