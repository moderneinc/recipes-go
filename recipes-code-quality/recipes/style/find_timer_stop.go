/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindTimerWithoutStop finds calls to `time.NewTimer` and `time.AfterFunc`.
// Timers should be stopped when no longer needed to release resources.
type FindTimerWithoutStop struct {
	recipe.Base
}

func (r *FindTimerWithoutStop) Name() string {
	return "org.openrewrite.golang.codequality.FindTimerWithoutStop"
}
func (r *FindTimerWithoutStop) DisplayName() string { return "Find timer without stop" }
func (r *FindTimerWithoutStop) Description() string {
	return "Find calls to `time.NewTimer` and `time.AfterFunc`. Timers should be stopped when no longer needed to release resources."
}
func (r *FindTimerWithoutStop) Tags() []string { return []string{"style", "resource-management"} }

func (r *FindTimerWithoutStop) Editor() recipe.TreeVisitor {
	return visitor.Init(&findTimerWithoutStopVisitor{})
}

type findTimerWithoutStopVisitor struct {
	visitor.GoVisitor
}

var timerMethods = map[string]bool{
	"NewTimer":  true,
	"AfterFunc": true,
}

func (v *findTimerWithoutStopVisitor) VisitMethodInvocation(mi *tree.MethodInvocation, p any) tree.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*tree.MethodInvocation)

	if mi.Select == nil {
		return mi
	}

	ident, ok := mi.Select.Element.(*tree.Identifier)
	if !ok || ident.Name != "time" {
		return mi
	}

	if !timerMethods[mi.Name.Name] {
		return mi
	}

	mi = mi.WithMarkers(tree.FoundSearchResult(mi.Markers, "ensure timer is stopped when no longer needed"))
	return mi
}
