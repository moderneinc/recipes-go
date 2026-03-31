/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package performance

import (
	"fmt"

	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/template"
)

var (
	siN = template.Expr("siN")

	preferStrconvItoaImpl = template.NewRecipe(
		template.RecipeName("org.openrewrite.golang.codequality.PreferStrconvItoa$Impl"),
		template.WithDisplayName("Prefer strconv.Itoa over fmt.Sprintf"),
		template.WithBefore(fmt.Sprintf(`fmt.Sprintf("%%d", %s)`, siN), template.Imports("fmt")),
		template.WithAfter(fmt.Sprintf(`strconv.Itoa(%s)`, siN), template.Imports("strconv")),
		template.WithCaptures(siN),
	)
)

// PreferStrconvItoa replaces `fmt.Sprintf("%d", n)` with `strconv.Itoa(n)`
// for better performance on int-to-string conversion.
type PreferStrconvItoa struct {
	recipe.Base
}

func (r *PreferStrconvItoa) Name() string {
	return "org.openrewrite.golang.codequality.PreferStrconvItoa"
}
func (r *PreferStrconvItoa) DisplayName() string {
	return "Prefer strconv.Itoa over fmt.Sprintf"
}
func (r *PreferStrconvItoa) Description() string {
	return "Replace `fmt.Sprintf(\"%d\", n)` with `strconv.Itoa(n)` for better performance."
}
func (r *PreferStrconvItoa) Tags() []string { return []string{"performance"} }

func (r *PreferStrconvItoa) RecipeList() []recipe.Recipe {
	return []recipe.Recipe{preferStrconvItoaImpl}
}
