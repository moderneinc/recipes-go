/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindContextTodo finds calls to `context.TODO()`. These are placeholders
// indicating that the proper context to use is not yet known and should
// eventually be replaced with a real context.
type FindContextTodo struct {
	recipe.Base
}

func (r *FindContextTodo) Name() string {
	return "org.openrewrite.golang.codequality.FindContextTodo"
}
func (r *FindContextTodo) DisplayName() string { return "Find context.TODO() calls" }
func (r *FindContextTodo) Description() string {
	return "Find calls to `context.TODO()`. These are placeholders that should eventually be replaced with a real context."
}
func (r *FindContextTodo) Tags() []string { return []string{"style"} }

func (r *FindContextTodo) Editor() recipe.TreeVisitor {
	return visitor.Init(&findContextTodoVisitor{})
}

type findContextTodoVisitor struct {
	visitor.GoVisitor
}

func (v *findContextTodoVisitor) VisitMethodInvocation(mi *tree.MethodInvocation, p any) tree.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*tree.MethodInvocation)

	if mi.Select == nil {
		return mi
	}

	ident, ok := mi.Select.Element.(*tree.Identifier)
	if !ok || ident.Name != "context" {
		return mi
	}

	if mi.Name.Name != "TODO" {
		return mi
	}

	mi = mi.WithMarkers(tree.FoundSearchResult(mi.Markers, "context.TODO() call; replace with a real context"))
	return mi
}
