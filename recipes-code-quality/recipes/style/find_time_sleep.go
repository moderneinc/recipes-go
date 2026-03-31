/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindTimeSleep finds calls to `time.Sleep()`. In production code, sleeping is
// often a code smell — consider using tickers, timers, context deadlines, or
// channel-based synchronization instead.
type FindTimeSleep struct {
	recipe.Base
}

func (r *FindTimeSleep) Name() string {
	return "org.openrewrite.golang.codequality.FindTimeSleep"
}
func (r *FindTimeSleep) DisplayName() string { return "Find time.Sleep calls" }
func (r *FindTimeSleep) Description() string {
	return "Find calls to `time.Sleep()`. In production code, sleeping is often a code smell — consider using tickers, timers, or context-based synchronization."
}
func (r *FindTimeSleep) Tags() []string { return []string{"style", "concurrency"} }

func (r *FindTimeSleep) Editor() recipe.TreeVisitor {
	return visitor.Init(&findTimeSleepVisitor{})
}

type findTimeSleepVisitor struct {
	visitor.GoVisitor
}

func (v *findTimeSleepVisitor) VisitMethodInvocation(mi *tree.MethodInvocation, p any) tree.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*tree.MethodInvocation)

	if mi.Select == nil {
		return mi
	}

	ident, ok := mi.Select.Element.(*tree.Identifier)
	if !ok || ident.Name != "time" {
		return mi
	}

	if mi.Name.Name != "Sleep" {
		return mi
	}

	mi = mi.WithMarkers(tree.FoundSearchResult(mi.Markers, "consider using tickers, timers, or context-based synchronization"))
	return mi
}
