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
	chrR = template.Expr("chrR")

	simplifySprintfCharImpl = template.NewRecipe(
		template.RecipeName("org.openrewrite.golang.codequality.SimplifySprintfChar$Impl"),
		template.WithDisplayName("Simplify fmt.Sprintf %c to string conversion"),
		template.WithBefore(fmt.Sprintf(`fmt.Sprintf("%%c", %s)`, chrR), template.Imports("fmt")),
		template.WithAfter(fmt.Sprintf(`string(%s)`, chrR)),
		template.WithCaptures(chrR),
	)
)

// SimplifySprintfChar replaces `fmt.Sprintf("%c", r)` with `string(r)` for
// better performance on rune-to-string conversion.
type SimplifySprintfChar struct {
	recipe.Base
}

func (r *SimplifySprintfChar) Name() string {
	return "org.openrewrite.golang.codequality.SimplifySprintfChar"
}
func (r *SimplifySprintfChar) DisplayName() string {
	return "Simplify fmt.Sprintf %c to string conversion"
}
func (r *SimplifySprintfChar) Description() string {
	return "Replace `fmt.Sprintf(\"%c\", r)` with `string(r)` for better performance."
}
func (r *SimplifySprintfChar) Tags() []string { return []string{"performance"} }

func (r *SimplifySprintfChar) RecipeList() []recipe.Recipe {
	return []recipe.Recipe{simplifySprintfCharImpl}
}
