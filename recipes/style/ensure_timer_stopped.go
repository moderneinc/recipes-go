/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// EnsureTimerStopped finds calls to `time.NewTimer` and inserts
// `defer timer.Stop()` after the assignment.
type EnsureTimerStopped struct {
	recipe.Base
}

func (r *EnsureTimerStopped) Name() string {
	return "org.openrewrite.golang.codequality.EnsureTimerStopped"
}
func (r *EnsureTimerStopped) DisplayName() string { return "Ensure timer stopped" }
func (r *EnsureTimerStopped) Description() string {
	return "Find calls to `time.NewTimer`. Timers should be stopped when no longer needed to release resources."
}
func (r *EnsureTimerStopped) Tags() []string { return []string{"style", "resource-management"} }

func (r *EnsureTimerStopped) Editor() recipe.TreeVisitor {
	return visitor.Init(&ensureTimerStoppedVisitor{})
}

type ensureTimerStoppedVisitor struct {
	visitor.GoVisitor
}

func (v *ensureTimerStoppedVisitor) VisitBlock(block *java.Block, p any) java.J {
	block = v.GoVisitor.VisitBlock(block, p).(*java.Block)
	return insertDeferMethodCall(block, isTimeNewTimer, "Stop")
}

// isTimeNewTimer covers time.NewTimer but not time.AfterFunc: the callback an
// AfterFunc timer holds is scheduled to run after the enclosing function
// returns, which is when a deferred Stop would cancel it.
func isTimeNewTimer(a acquisition) bool {
	declaring, ok := declaringType(a.call)
	return ok && declaring == "time" && a.call.Name.Name == "NewTimer"
}
