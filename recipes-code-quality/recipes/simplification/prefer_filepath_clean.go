/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification

import (
	"fmt"

	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/template"
)

var fcpP = template.Expr("fcpP")

// PreferFilepathClean replaces `filepath.Join(filepath.Clean(p))` with
// `filepath.Clean(p)`. Wrapping a single Clean result in Join is redundant
// because Join with one argument simply returns Clean of that argument, so
// the outer Join adds no value.
type PreferFilepathClean struct {
	recipe.Base
}

func (r *PreferFilepathClean) Name() string {
	return "org.openrewrite.golang.codequality.PreferFilepathClean"
}
func (r *PreferFilepathClean) DisplayName() string {
	return "Prefer filepath.Clean over redundant filepath.Join"
}
func (r *PreferFilepathClean) Description() string {
	return "Replace `filepath.Join(filepath.Clean(p))` with `filepath.Clean(p)` since Join with a single already-cleaned argument is redundant."
}
func (r *PreferFilepathClean) Tags() []string { return []string{"cleanup", "simplification"} }

var preferFilepathCleanImpl = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.PreferFilepathClean$Impl"),
	template.WithDisplayName("filepath.Join(filepath.Clean) → filepath.Clean"),
	template.WithBefore(fmt.Sprintf(`filepath.Join(filepath.Clean(%s))`, fcpP), template.Imports("path/filepath")),
	template.WithAfter(fmt.Sprintf(`filepath.Clean(%s)`, fcpP), template.Imports("path/filepath")),
	template.WithCaptures(fcpP),
)

func (r *PreferFilepathClean) RecipeList() []recipe.Recipe {
	return []recipe.Recipe{preferFilepathCleanImpl}
}
