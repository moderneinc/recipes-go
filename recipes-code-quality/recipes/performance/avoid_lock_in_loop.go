/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package performance

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
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

func (v *avoidLockInLoopVisitor) VisitForLoop(forLoop *java.ForLoop, p any) java.J {
	v.insideLoop++
	forLoop = v.GoVisitor.VisitForLoop(forLoop, p).(*java.ForLoop)
	v.insideLoop--
	return forLoop
}

func (v *avoidLockInLoopVisitor) VisitForEachLoop(forEach *java.ForEachLoop, p any) java.J {
	v.insideLoop++
	forEach = v.GoVisitor.VisitForEachLoop(forEach, p).(*java.ForEachLoop)
	v.insideLoop--
	return forEach
}

func (v *avoidLockInLoopVisitor) VisitMethodInvocation(mi *java.MethodInvocation, p any) java.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*java.MethodInvocation)

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
		java.MarkupWarn(mi.Markers, "lock acquisition in loop; consider locking once outside the loop"),
	)
	return mi
}
