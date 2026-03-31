/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindHTTPListenAndServe finds calls to `http.ListenAndServe` and
// `http.ListenAndServeTLS`. The default http.Server has no read, write, or idle
// timeouts, which makes the server vulnerable to denial-of-service attacks.
// Use an explicit `http.Server` with timeouts configured instead.
type FindHTTPListenAndServe struct {
	recipe.Base
}

func (r *FindHTTPListenAndServe) Name() string {
	return "org.openrewrite.golang.codequality.FindHTTPListenAndServe"
}
func (r *FindHTTPListenAndServe) DisplayName() string {
	return "Find http.ListenAndServe without timeouts"
}
func (r *FindHTTPListenAndServe) Description() string {
	return "Find calls to `http.ListenAndServe` and `http.ListenAndServeTLS`. The default server has no timeouts, which is a denial-of-service risk."
}
func (r *FindHTTPListenAndServe) Tags() []string { return []string{"security"} }

func (r *FindHTTPListenAndServe) Editor() recipe.TreeVisitor {
	return visitor.Init(&findHTTPListenAndServeVisitor{})
}

type findHTTPListenAndServeVisitor struct {
	visitor.GoVisitor
}

func (v *findHTTPListenAndServeVisitor) VisitMethodInvocation(mi *tree.MethodInvocation, p any) tree.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*tree.MethodInvocation)

	if mi.Select == nil {
		return mi
	}

	ident, ok := mi.Select.Element.(*tree.Identifier)
	if !ok || ident.Name != "http" {
		return mi
	}

	if mi.Name.Name != "ListenAndServe" && mi.Name.Name != "ListenAndServeTLS" {
		return mi
	}

	mi = mi.WithMarkers(tree.FoundSearchResult(mi.Markers, "http.ListenAndServe has no timeouts; use an http.Server with timeouts"))
	return mi
}
