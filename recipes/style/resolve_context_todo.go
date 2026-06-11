/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/template"
)

// ResolveContextTodo replaces calls to `context.TODO()` with
// `context.Background()`. These are placeholders indicating that the proper
// context to use is not yet known; `context.Background()` is the safe default.
type ResolveContextTodo struct {
	recipe.Base
}

func (r *ResolveContextTodo) Name() string {
	return "org.openrewrite.golang.codequality.ResolveContextTodo"
}
func (r *ResolveContextTodo) DisplayName() string { return "Resolve context.TODO" }
func (r *ResolveContextTodo) Description() string {
	return "Replace `context.TODO()` with `context.Background()`. These are placeholders that should be replaced with a real context."
}
func (r *ResolveContextTodo) Tags() []string { return []string{"style"} }

var resolveContextTodoImpl = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.ResolveContextTodo$Impl"),
	template.WithDisplayName("context.TODO() -> context.Background()"),
	template.WithBefore(`context.TODO()`, template.Imports("context")),
	template.WithAfter(`context.Background()`, template.Imports("context")),
)

func (r *ResolveContextTodo) RecipeList() []recipe.Recipe {
	return []recipe.Recipe{resolveContextTodoImpl}
}
