/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package performance

import (
	"fmt"

	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/template"
)

var (
	fbB = template.Expr("fbB")

	preferStrconvFormatBoolImpl = template.NewRecipe(
		template.RecipeName("org.openrewrite.golang.codequality.PreferStrconvFormatBool$Impl"),
		template.WithDisplayName("Prefer strconv.FormatBool over fmt.Sprintf"),
		template.WithBefore(fmt.Sprintf(`fmt.Sprintf("%%t", %s)`, fbB), template.Imports("fmt")),
		template.WithAfter(fmt.Sprintf(`strconv.FormatBool(%s)`, fbB), template.Imports("strconv")),
		template.WithCaptures(fbB),
	)
)

// PreferStrconvFormatBool replaces `fmt.Sprintf("%t", b)` with
// `strconv.FormatBool(b)` for better performance on bool-to-string conversion.
type PreferStrconvFormatBool struct {
	recipe.Base
}

func (r *PreferStrconvFormatBool) Name() string {
	return "org.openrewrite.golang.codequality.PreferStrconvFormatBool"
}
func (r *PreferStrconvFormatBool) DisplayName() string {
	return "Prefer strconv.FormatBool over fmt.Sprintf"
}
func (r *PreferStrconvFormatBool) Description() string {
	return "Replace `fmt.Sprintf(\"%t\", b)` with `strconv.FormatBool(b)` for better performance."
}
func (r *PreferStrconvFormatBool) Tags() []string { return []string{"performance"} }

func (r *PreferStrconvFormatBool) RecipeList() []recipe.Recipe {
	return []recipe.Recipe{preferStrconvFormatBoolImpl}
}
