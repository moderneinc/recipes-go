/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindContextBackground finds calls to `context.Background()`. In most
// application code the context should be propagated from the caller rather
// than creating a new empty context via `context.Background()`.
type FindContextBackground struct {
	recipe.Base
}

func (r *FindContextBackground) Name() string {
	return "org.openrewrite.golang.codequality.FindContextBackground"
}
func (r *FindContextBackground) DisplayName() string { return "Find context.Background() calls" }
func (r *FindContextBackground) Description() string {
	return "Find calls to `context.Background()`. Consider using a context passed from the caller instead."
}
func (r *FindContextBackground) Tags() []string { return []string{"style"} }

func (r *FindContextBackground) Editor() recipe.TreeVisitor {
	return visitor.Init(&findContextBackgroundVisitor{})
}

type findContextBackgroundVisitor struct {
	visitor.GoVisitor
}

func (v *findContextBackgroundVisitor) VisitMethodInvocation(mi *tree.MethodInvocation, p any) tree.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*tree.MethodInvocation)

	if mi.Select == nil {
		return mi
	}

	ident, ok := mi.Select.Element.(*tree.Identifier)
	if !ok || ident.Name != "context" {
		return mi
	}

	if mi.Name.Name != "Background" {
		return mi
	}

	mi = mi.WithMarkers(tree.FoundSearchResult(mi.Markers, "context.Background() call; consider using a passed context instead"))
	return mi
}
