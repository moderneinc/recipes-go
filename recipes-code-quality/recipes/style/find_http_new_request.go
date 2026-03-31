/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindHttpNewRequest finds calls to `http.NewRequest()`. Prefer
// `http.NewRequestWithContext()` to propagate cancellation and deadlines
// through the request lifecycle.
type FindHttpNewRequest struct {
	recipe.Base
}

func (r *FindHttpNewRequest) Name() string {
	return "org.openrewrite.golang.codequality.FindHttpNewRequest"
}
func (r *FindHttpNewRequest) DisplayName() string { return "Find http.NewRequest calls" }
func (r *FindHttpNewRequest) Description() string {
	return "Find calls to `http.NewRequest`. Consider using `http.NewRequestWithContext` to propagate cancellation and deadlines."
}
func (r *FindHttpNewRequest) Tags() []string { return []string{"style", "net/http"} }

func (r *FindHttpNewRequest) Editor() recipe.TreeVisitor {
	return visitor.Init(&findHttpNewRequestVisitor{})
}

type findHttpNewRequestVisitor struct {
	visitor.GoVisitor
}

func (v *findHttpNewRequestVisitor) VisitMethodInvocation(mi *tree.MethodInvocation, p any) tree.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*tree.MethodInvocation)

	if mi.Select == nil {
		return mi
	}

	ident, ok := mi.Select.Element.(*tree.Identifier)
	if !ok || ident.Name != "http" {
		return mi
	}

	if mi.Name.Name != "NewRequest" {
		return mi
	}

	mi = mi.WithMarkers(tree.FoundSearchResult(mi.Markers, "consider using http.NewRequestWithContext"))
	return mi
}
