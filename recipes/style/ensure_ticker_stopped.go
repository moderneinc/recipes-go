/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// EnsureTickerStopped finds calls to `time.NewTicker` and inserts
// `defer ticker.Stop()` after the assignment.
type EnsureTickerStopped struct {
	recipe.Base
}

func (r *EnsureTickerStopped) Name() string {
	return "org.openrewrite.golang.codequality.EnsureTickerStopped"
}
func (r *EnsureTickerStopped) DisplayName() string { return "Ensure ticker stopped" }
func (r *EnsureTickerStopped) Description() string {
	return "Find calls to `time.NewTicker`. Tickers must be stopped when no longer needed to avoid goroutine leaks."
}
func (r *EnsureTickerStopped) Tags() []string { return []string{"style", "resource-management"} }

func (r *EnsureTickerStopped) Editor() recipe.TreeVisitor {
	return visitor.Init(&ensureTickerStoppedVisitor{})
}

type ensureTickerStoppedVisitor struct {
	visitor.GoVisitor
}

func (v *ensureTickerStoppedVisitor) VisitBlock(block *java.Block, p any) java.J {
	block = v.GoVisitor.VisitBlock(block, p).(*java.Block)
	return insertDeferMethodCall(block, isTimeNewTicker, "Stop")
}

// isTimeNewTicker returns true if the method invocation is time.NewTicker.
func isTimeNewTicker(a acquisition) bool {
	declaring, ok := declaringType(a.call)
	return ok && declaring == "time" && a.call.Name.Name == "NewTicker"
}
