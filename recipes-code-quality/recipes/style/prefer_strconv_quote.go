/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"fmt"

	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/template"
)

var sqS = template.Expr("sqS")

var preferStrconvQuoteImpl = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.PreferStrconvQuote$Impl"),
	template.WithDisplayName("fmt.Sprintf(\"%q\", s) → strconv.Quote(s)"),
	template.WithBefore(fmt.Sprintf(`fmt.Sprintf("%%q", %s)`, sqS), template.Imports("fmt")),
	template.WithAfter(fmt.Sprintf(`strconv.Quote(%s)`, sqS), template.Imports("strconv")),
	template.WithCaptures(sqS),
)

// PreferStrconvQuote replaces `fmt.Sprintf("%q", s)` with `strconv.Quote(s)`
// for clearer intent when quoting strings.
type PreferStrconvQuote struct {
	recipe.Base
}

func (r *PreferStrconvQuote) Name() string {
	return "org.openrewrite.golang.codequality.PreferStrconvQuote"
}
func (r *PreferStrconvQuote) DisplayName() string {
	return "Prefer strconv.Quote over fmt.Sprintf"
}
func (r *PreferStrconvQuote) Description() string {
	return "Replace `fmt.Sprintf(\"%q\", s)` with `strconv.Quote(s)` for clearer intent when quoting strings."
}
func (r *PreferStrconvQuote) Tags() []string { return []string{"style", "cleanup"} }

func (r *PreferStrconvQuote) RecipeList() []recipe.Recipe {
	return []recipe.Recipe{preferStrconvQuoteImpl}
}
