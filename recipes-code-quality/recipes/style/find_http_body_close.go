/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindHttpResponseBody finds calls to `http.Get`, `http.Post`, `http.Head`,
// and `client.Do` whose response body must be closed to avoid resource leaks.
type FindHttpResponseBody struct {
	recipe.Base
}

func (r *FindHttpResponseBody) Name() string {
	return "org.openrewrite.golang.codequality.FindHttpResponseBody"
}
func (r *FindHttpResponseBody) DisplayName() string { return "Find HTTP response body that must be closed" }
func (r *FindHttpResponseBody) Description() string {
	return "Find calls to `http.Get`, `http.Post`, `http.Head`, and `client.Do` whose response body must be closed to avoid resource leaks."
}
func (r *FindHttpResponseBody) Tags() []string { return []string{"style", "resource-management"} }

func (r *FindHttpResponseBody) Editor() recipe.TreeVisitor {
	return visitor.Init(&findHttpResponseBodyVisitor{})
}

type findHttpResponseBodyVisitor struct {
	visitor.GoVisitor
}

// httpBodyMethods lists the net/http convenience functions whose response
// body must always be closed by the caller.
var httpBodyMethods = map[string]bool{
	"Get":  true,
	"Post": true,
	"Head": true,
}

func (v *findHttpResponseBodyVisitor) VisitMethodInvocation(mi *tree.MethodInvocation, p any) tree.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*tree.MethodInvocation)

	if mi.Select == nil {
		return mi
	}

	ident, ok := mi.Select.Element.(*tree.Identifier)
	if !ok {
		return mi
	}

	// Match http.Get / http.Post / http.Head
	if ident.Name == "http" && httpBodyMethods[mi.Name.Name] {
		mi = mi.WithMarkers(tree.FoundSearchResult(mi.Markers, "ensure response body is closed"))
		return mi
	}

	// Match client.Do (any receiver calling Do)
	if mi.Name.Name == "Do" {
		mi = mi.WithMarkers(tree.FoundSearchResult(mi.Markers, "ensure response body is closed"))
		return mi
	}

	return mi
}
