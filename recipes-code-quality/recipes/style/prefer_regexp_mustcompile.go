/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"fmt"

	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/template"
)

var regexpPattern = template.Expr("p")

// PreferRegexpMustCompile replaces `regexp.Compile(p)` in package-level var
// declarations with `regexp.MustCompile(p)`. At package level, a compilation
// error should panic rather than be silently ignored.
type PreferRegexpMustCompile struct {
	recipe.Base
}

func (r *PreferRegexpMustCompile) Name() string {
	return "org.openrewrite.golang.codequality.PreferRegexpMustCompile"
}
func (r *PreferRegexpMustCompile) DisplayName() string {
	return "Prefer regexp.MustCompile at package level"
}
func (r *PreferRegexpMustCompile) Description() string {
	return "Replace `regexp.Compile(p)` with `regexp.MustCompile(p)` in package-level var declarations."
}
func (r *PreferRegexpMustCompile) Tags() []string { return []string{"style"} }

var preferRegexpMustCompileImpl = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.PreferRegexpMustCompile$Impl"),
	template.WithDisplayName("regexp.Compile → regexp.MustCompile"),
	template.WithBefore(fmt.Sprintf(`regexp.Compile(%s)`, regexpPattern), template.Imports("regexp")),
	template.WithAfter(fmt.Sprintf(`regexp.MustCompile(%s)`, regexpPattern), template.Imports("regexp")),
	template.WithCaptures(regexpPattern),
)

func (r *PreferRegexpMustCompile) RecipeList() []recipe.Recipe {
	return []recipe.Recipe{preferRegexpMustCompileImpl}
}
