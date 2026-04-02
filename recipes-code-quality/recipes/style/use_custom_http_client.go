/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// httpDefaultClientMethods lists the top-level net/http convenience functions
// that use the default client (which has no timeout configured).
var httpDefaultClientMethods = map[string]bool{
	"Get":      true,
	"Post":     true,
	"Head":     true,
	"PostForm": true,
}

// UseCustomHttpClient finds calls to `http.Get`, `http.Post`, `http.Head`,
// and `http.PostForm`. These convenience functions use the default HTTP client
// which has no timeout configured, potentially leading to resource leaks.
type UseCustomHttpClient struct {
	recipe.Base
}

func (r *UseCustomHttpClient) Name() string {
	return "org.openrewrite.golang.codequality.UseCustomHttpClient"
}
func (r *UseCustomHttpClient) DisplayName() string { return "Use custom HTTP client" }
func (r *UseCustomHttpClient) Description() string {
	return "Find calls to `http.Get`, `http.Post`, `http.Head`, and `http.PostForm` which use the default HTTP client without a timeout."
}
func (r *UseCustomHttpClient) Tags() []string { return []string{"style", "net/http"} }

func (r *UseCustomHttpClient) Editor() recipe.TreeVisitor {
	return visitor.Init(&useCustomHttpClientVisitor{})
}

type useCustomHttpClientVisitor struct {
	visitor.GoVisitor
}

func (v *useCustomHttpClientVisitor) VisitMethodInvocation(mi *tree.MethodInvocation, p any) tree.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*tree.MethodInvocation)

	if mi.Select == nil {
		return mi
	}

	ident, ok := mi.Select.Element.(*tree.Identifier)
	if !ok || ident.Name != "http" {
		return mi
	}

	if !httpDefaultClientMethods[mi.Name.Name] {
		return mi
	}

	mi = mi.WithMarkers(tree.MarkupWarn(mi.Markers, "uses default HTTP client without timeout"))
	return mi
}
