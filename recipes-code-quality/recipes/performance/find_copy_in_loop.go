/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package performance

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindCopyInLoop finds `copy()` calls inside for/range loops. Repeated
// copying in loops may indicate a buffer reuse opportunity.
type FindCopyInLoop struct {
	recipe.Base
}

func (r *FindCopyInLoop) Name() string {
	return "org.openrewrite.golang.codequality.FindCopyInLoop"
}
func (r *FindCopyInLoop) DisplayName() string { return "Find copy in loop" }
func (r *FindCopyInLoop) Description() string {
	return "Find `copy()` calls inside for/range loops. Repeated copying in loops may indicate a buffer reuse opportunity."
}
func (r *FindCopyInLoop) Tags() []string { return []string{"performance"} }

func (r *FindCopyInLoop) Editor() recipe.TreeVisitor {
	return visitor.Init(&findCopyInLoopVisitor{})
}

type findCopyInLoopVisitor struct {
	visitor.GoVisitor
	insideLoop int
}

func (v *findCopyInLoopVisitor) VisitForLoop(forLoop *tree.ForLoop, p any) tree.J {
	v.insideLoop++
	forLoop = v.GoVisitor.VisitForLoop(forLoop, p).(*tree.ForLoop)
	v.insideLoop--
	return forLoop
}

func (v *findCopyInLoopVisitor) VisitForEachLoop(forEach *tree.ForEachLoop, p any) tree.J {
	v.insideLoop++
	forEach = v.GoVisitor.VisitForEachLoop(forEach, p).(*tree.ForEachLoop)
	v.insideLoop--
	return forEach
}

func (v *findCopyInLoopVisitor) VisitMethodInvocation(mi *tree.MethodInvocation, p any) tree.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*tree.MethodInvocation)

	if v.insideLoop == 0 {
		return mi
	}

	// Match: copy(dst, src) — built-in, so no Select and Name == "copy".
	if mi.Select != nil || mi.Name.Name != "copy" {
		return mi
	}

	mi = mi.WithMarkers(
		tree.FoundSearchResult(mi.Markers, "copy in loop; consider reusing buffer outside loop"),
	)
	return mi
}
