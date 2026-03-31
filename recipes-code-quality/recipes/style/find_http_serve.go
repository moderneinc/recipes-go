/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindHttpListenAndServe finds calls to `http.ListenAndServe` which serve
// traffic without TLS. Consider using `http.ListenAndServeTLS` instead to
// encrypt traffic in transit.
type FindHttpListenAndServe struct {
	recipe.Base
}

func (r *FindHttpListenAndServe) Name() string {
	return "org.openrewrite.golang.codequality.FindHttpListenAndServe"
}
func (r *FindHttpListenAndServe) DisplayName() string {
	return "Find http.ListenAndServe without TLS"
}
func (r *FindHttpListenAndServe) Description() string {
	return "Find calls to `http.ListenAndServe` which serve traffic without TLS. Consider using `http.ListenAndServeTLS` instead."
}
func (r *FindHttpListenAndServe) Tags() []string { return []string{"security"} }

func (r *FindHttpListenAndServe) Editor() recipe.TreeVisitor {
	return visitor.Init(&findHttpListenAndServeVisitor{})
}

type findHttpListenAndServeVisitor struct {
	visitor.GoVisitor
}

func (v *findHttpListenAndServeVisitor) VisitMethodInvocation(mi *tree.MethodInvocation, p any) tree.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*tree.MethodInvocation)

	if mi.Select == nil {
		return mi
	}

	ident, ok := mi.Select.Element.(*tree.Identifier)
	if !ok || ident.Name != "http" {
		return mi
	}

	if mi.Name.Name != "ListenAndServe" {
		return mi
	}

	mi = mi.WithMarkers(tree.FoundSearchResult(mi.Markers, "http.ListenAndServe without TLS; consider http.ListenAndServeTLS"))
	return mi
}
