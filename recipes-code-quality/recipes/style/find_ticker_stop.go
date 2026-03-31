/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindTickerWithoutStop finds calls to `time.NewTicker`. Tickers must be
// stopped when no longer needed to avoid goroutine leaks.
type FindTickerWithoutStop struct {
	recipe.Base
}

func (r *FindTickerWithoutStop) Name() string {
	return "org.openrewrite.golang.codequality.FindTickerWithoutStop"
}
func (r *FindTickerWithoutStop) DisplayName() string { return "Find ticker without stop" }
func (r *FindTickerWithoutStop) Description() string {
	return "Find calls to `time.NewTicker`. Tickers must be stopped when no longer needed to avoid goroutine leaks."
}
func (r *FindTickerWithoutStop) Tags() []string { return []string{"style", "resource-management"} }

func (r *FindTickerWithoutStop) Editor() recipe.TreeVisitor {
	return visitor.Init(&findTickerWithoutStopVisitor{})
}

type findTickerWithoutStopVisitor struct {
	visitor.GoVisitor
}

func (v *findTickerWithoutStopVisitor) VisitMethodInvocation(mi *tree.MethodInvocation, p any) tree.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*tree.MethodInvocation)

	if mi.Select == nil {
		return mi
	}

	ident, ok := mi.Select.Element.(*tree.Identifier)
	if !ok || ident.Name != "time" {
		return mi
	}

	if mi.Name.Name != "NewTicker" {
		return mi
	}

	mi = mi.WithMarkers(tree.FoundSearchResult(mi.Markers, "ensure ticker is stopped to avoid goroutine leaks"))
	return mi
}
