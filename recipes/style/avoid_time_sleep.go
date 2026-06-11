/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// AvoidTimeSleep finds calls to `time.Sleep()`. In production code, sleeping is
// often a code smell — consider using tickers, timers, context deadlines, or
// channel-based synchronization instead.
type AvoidTimeSleep struct {
	recipe.Base
}

func (r *AvoidTimeSleep) Name() string {
	return "org.openrewrite.golang.codequality.AvoidTimeSleep"
}
func (r *AvoidTimeSleep) DisplayName() string { return "Avoid time.Sleep" }
func (r *AvoidTimeSleep) Description() string {
	return "Find calls to `time.Sleep()`. In production code, sleeping is often a code smell — consider using tickers, timers, or context-based synchronization."
}
func (r *AvoidTimeSleep) Tags() []string { return []string{"style", "concurrency"} }

func (r *AvoidTimeSleep) Editor() recipe.TreeVisitor {
	return visitor.Init(&avoidTimeSleepVisitor{})
}

type avoidTimeSleepVisitor struct {
	visitor.GoVisitor
}

func (v *avoidTimeSleepVisitor) VisitMethodInvocation(mi *java.MethodInvocation, p any) java.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*java.MethodInvocation)

	if mi.Select == nil {
		return mi
	}

	ident, ok := mi.Select.Element.(*java.Identifier)
	if !ok || ident.Name != "time" {
		return mi
	}

	if mi.Name.Name != "Sleep" {
		return mi
	}

	mi = mi.WithMarkers(java.MarkupInfo(mi.Markers, "consider using tickers, timers, or context-based synchronization"))
	return mi
}
