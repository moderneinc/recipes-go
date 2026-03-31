/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling

import (
	"fmt"

	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/template"
)

var ejErr = template.Expr("ejErr")

var simplifyRedundantErrorWrapImpl = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.SimplifyRedundantErrorWrap$Impl"),
	template.WithDisplayName("Simplify redundant error wrap"),
	template.WithBefore(fmt.Sprintf(`fmt.Errorf("%%w", %s)`, ejErr), template.Imports("fmt")),
	template.WithAfter(fmt.Sprintf(`%s`, ejErr)),
	template.WithCaptures(ejErr),
)

// SimplifyRedundantErrorWrap replaces `fmt.Errorf("%w", err)` with just `err`.
// Wrapping an error with no additional context is redundant.
type SimplifyRedundantErrorWrap struct {
	recipe.Base
}

func (r *SimplifyRedundantErrorWrap) Name() string {
	return "org.openrewrite.golang.codequality.SimplifyRedundantErrorWrap"
}
func (r *SimplifyRedundantErrorWrap) DisplayName() string { return "Simplify redundant error wrap" }
func (r *SimplifyRedundantErrorWrap) Description() string {
	return "Replace `fmt.Errorf(\"%w\", err)` with `err` when wrapping adds no context."
}
func (r *SimplifyRedundantErrorWrap) Tags() []string { return []string{"error-handling"} }

func (r *SimplifyRedundantErrorWrap) RecipeList() []recipe.Recipe {
	return []recipe.Recipe{simplifyRedundantErrorWrapImpl}
}
