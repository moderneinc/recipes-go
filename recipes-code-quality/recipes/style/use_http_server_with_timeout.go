/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// UseHttpServerWithTimeout finds calls to `http.ListenAndServe` and
// `http.ListenAndServeTLS`. The default http.Server has no read, write, or idle
// timeouts, which makes the server vulnerable to denial-of-service attacks.
// Use an explicit `http.Server` with timeouts configured instead.
type UseHttpServerWithTimeout struct {
	recipe.Base
}

func (r *UseHttpServerWithTimeout) Name() string {
	return "org.openrewrite.golang.codequality.UseHttpServerWithTimeout"
}
func (r *UseHttpServerWithTimeout) DisplayName() string {
	return "Use http.Server with timeouts"
}
func (r *UseHttpServerWithTimeout) Description() string {
	return "Find calls to `http.ListenAndServe` and `http.ListenAndServeTLS`. The default server has no timeouts, which is a denial-of-service risk."
}
func (r *UseHttpServerWithTimeout) Tags() []string { return []string{"security"} }

func (r *UseHttpServerWithTimeout) Editor() recipe.TreeVisitor {
	return visitor.Init(&useHttpServerWithTimeoutVisitor{})
}

type useHttpServerWithTimeoutVisitor struct {
	visitor.GoVisitor
}

func (v *useHttpServerWithTimeoutVisitor) VisitMethodInvocation(mi *tree.MethodInvocation, p any) tree.J {
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

	mi = mi.WithMarkers(tree.MarkupWarn(mi.Markers, "http.ListenAndServe has no timeouts; use an http.Server with timeouts"))
	return mi
}
