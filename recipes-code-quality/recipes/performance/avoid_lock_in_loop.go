/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package performance

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// AvoidLockInLoop finds `mu.Lock()` or `mu.RLock()` calls inside for/range
// loops. Acquiring locks in tight loops can cause contention and degrade
// performance; consider locking once outside the loop.
type AvoidLockInLoop struct {
	recipe.Base
}

func (r *AvoidLockInLoop) Name() string {
	return "org.openrewrite.golang.codequality.AvoidLockInLoop"
}
func (r *AvoidLockInLoop) DisplayName() string { return "Avoid lock in loop" }
func (r *AvoidLockInLoop) Description() string {
	return "Find `Lock()` or `RLock()` calls inside for/range loops. Acquiring locks in tight loops can cause contention; consider locking once outside the loop."
}
func (r *AvoidLockInLoop) Tags() []string { return []string{"performance"} }

func (r *AvoidLockInLoop) Editor() recipe.TreeVisitor {
	return visitor.Init(&avoidLockInLoopVisitor{})
}

type avoidLockInLoopVisitor struct {
	visitor.GoVisitor
	insideLoop int
}

func (v *avoidLockInLoopVisitor) VisitForLoop(forLoop *tree.ForLoop, p any) tree.J {
	v.insideLoop++
	forLoop = v.GoVisitor.VisitForLoop(forLoop, p).(*tree.ForLoop)
	v.insideLoop--
	return forLoop
}

func (v *avoidLockInLoopVisitor) VisitForEachLoop(forEach *tree.ForEachLoop, p any) tree.J {
	v.insideLoop++
	forEach = v.GoVisitor.VisitForEachLoop(forEach, p).(*tree.ForEachLoop)
	v.insideLoop--
	return forEach
}

func (v *avoidLockInLoopVisitor) VisitMethodInvocation(mi *tree.MethodInvocation, p any) tree.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*tree.MethodInvocation)

	if v.insideLoop == 0 {
		return mi
	}

	// Must have a receiver (e.g. mu.Lock()).
	if mi.Select == nil {
		return mi
	}

	if mi.Name.Name != "Lock" && mi.Name.Name != "RLock" {
		return mi
	}

	mi = mi.WithMarkers(
		tree.MarkupWarn(mi.Markers, "lock acquisition in loop; consider locking once outside the loop"),
	)
	return mi
}
