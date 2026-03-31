/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindHttpRedirect finds calls to `http.Redirect`. These calls should be
// reviewed to ensure redirect targets are validated and status codes are
// appropriate.
type FindHttpRedirect struct {
	recipe.Base
}

func (r *FindHttpRedirect) Name() string {
	return "org.openrewrite.golang.codequality.FindHttpRedirect"
}
func (r *FindHttpRedirect) DisplayName() string { return "Find HTTP redirects" }
func (r *FindHttpRedirect) Description() string {
	return "Find calls to `http.Redirect`. Review redirect targets to ensure they are validated and status codes are appropriate."
}
func (r *FindHttpRedirect) Tags() []string { return []string{"style", "net/http"} }

func (r *FindHttpRedirect) Editor() recipe.TreeVisitor {
	return visitor.Init(&findHttpRedirectVisitor{})
}

type findHttpRedirectVisitor struct {
	visitor.GoVisitor
}

func (v *findHttpRedirectVisitor) VisitMethodInvocation(mi *tree.MethodInvocation, p any) tree.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*tree.MethodInvocation)

	if mi.Select == nil {
		return mi
	}

	ident, ok := mi.Select.Element.(*tree.Identifier)
	if !ok || ident.Name != "http" {
		return mi
	}

	if mi.Name.Name != "Redirect" {
		return mi
	}

	mi = mi.WithMarkers(tree.FoundSearchResult(mi.Markers, "review redirect target and status code"))
	return mi
}
