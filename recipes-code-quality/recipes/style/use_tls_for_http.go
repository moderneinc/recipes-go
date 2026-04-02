/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// UseTlsForHttp finds calls to `http.ListenAndServe` which serve
// traffic without TLS. Consider using `http.ListenAndServeTLS` instead to
// encrypt traffic in transit.
type UseTlsForHttp struct {
	recipe.Base
}

func (r *UseTlsForHttp) Name() string {
	return "org.openrewrite.golang.codequality.UseTlsForHttp"
}
func (r *UseTlsForHttp) DisplayName() string {
	return "Use TLS for HTTP"
}
func (r *UseTlsForHttp) Description() string {
	return "Find calls to `http.ListenAndServe` which serve traffic without TLS. Consider using `http.ListenAndServeTLS` instead."
}
func (r *UseTlsForHttp) Tags() []string { return []string{"security"} }

func (r *UseTlsForHttp) Editor() recipe.TreeVisitor {
	return visitor.Init(&useTlsForHttpVisitor{})
}

type useTlsForHttpVisitor struct {
	visitor.GoVisitor
}

func (v *useTlsForHttpVisitor) VisitMethodInvocation(mi *tree.MethodInvocation, p any) tree.J {
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

	mi = mi.WithMarkers(tree.MarkupInfo(mi.Markers, "http.ListenAndServe without TLS; consider http.ListenAndServeTLS"))
	return mi
}
