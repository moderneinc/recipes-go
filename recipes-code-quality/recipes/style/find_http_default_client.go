/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// httpDefaultClientMethods lists the top-level net/http convenience functions
// that use the default client (which has no timeout configured).
var httpDefaultClientMethods = map[string]bool{
	"Get":      true,
	"Post":     true,
	"Head":     true,
	"PostForm": true,
}

// FindHttpDefaultClient finds calls to `http.Get`, `http.Post`, `http.Head`,
// and `http.PostForm`. These convenience functions use the default HTTP client
// which has no timeout configured, potentially leading to resource leaks.
type FindHttpDefaultClient struct {
	recipe.Base
}

func (r *FindHttpDefaultClient) Name() string {
	return "org.openrewrite.golang.codequality.FindHttpDefaultClient"
}
func (r *FindHttpDefaultClient) DisplayName() string { return "Find default HTTP client usage" }
func (r *FindHttpDefaultClient) Description() string {
	return "Find calls to `http.Get`, `http.Post`, `http.Head`, and `http.PostForm` which use the default HTTP client without a timeout."
}
func (r *FindHttpDefaultClient) Tags() []string { return []string{"style", "net/http"} }

func (r *FindHttpDefaultClient) Editor() recipe.TreeVisitor {
	return visitor.Init(&findHttpDefaultClientVisitor{})
}

type findHttpDefaultClientVisitor struct {
	visitor.GoVisitor
}

func (v *findHttpDefaultClientVisitor) VisitMethodInvocation(mi *tree.MethodInvocation, p any) tree.J {
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

	mi = mi.WithMarkers(tree.FoundSearchResult(mi.Markers, "uses default HTTP client without timeout"))
	return mi
}
