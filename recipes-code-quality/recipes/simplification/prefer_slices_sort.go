/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification

import (
	"fmt"

	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/template"
)

var ssrtS = template.Expr("ssrtS")

// PreferSlicesSort replaces deprecated `sort.Ints(s)`, `sort.Strings(s)`, and
// `sort.Float64s(s)` with the generic `slices.Sort(s)` from Go 1.21+.
type PreferSlicesSort struct {
	recipe.Base
}

func (r *PreferSlicesSort) Name() string {
	return "org.openrewrite.golang.codequality.PreferSlicesSort"
}
func (r *PreferSlicesSort) DisplayName() string { return "Prefer slices.Sort over sort type helpers" }
func (r *PreferSlicesSort) Description() string {
	return "Replace deprecated `sort.Ints`, `sort.Strings`, and `sort.Float64s` with `slices.Sort` (Go 1.21+)."
}
func (r *PreferSlicesSort) Tags() []string { return []string{"cleanup", "simplification"} }

var preferSlicesSortInts = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.PreferSlicesSort$Ints"),
	template.WithDisplayName("sort.Ints -> slices.Sort"),
	template.WithBefore(fmt.Sprintf(`sort.Ints(%s)`, ssrtS), template.Imports("sort")),
	template.WithAfter(fmt.Sprintf(`slices.Sort(%s)`, ssrtS), template.Imports("slices")),
	template.WithCaptures(ssrtS),
)

var preferSlicesSortStrings = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.PreferSlicesSort$Strings"),
	template.WithDisplayName("sort.Strings -> slices.Sort"),
	template.WithBefore(fmt.Sprintf(`sort.Strings(%s)`, ssrtS), template.Imports("sort")),
	template.WithAfter(fmt.Sprintf(`slices.Sort(%s)`, ssrtS), template.Imports("slices")),
	template.WithCaptures(ssrtS),
)

var preferSlicesSortFloat64s = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.PreferSlicesSort$Float64s"),
	template.WithDisplayName("sort.Float64s -> slices.Sort"),
	template.WithBefore(fmt.Sprintf(`sort.Float64s(%s)`, ssrtS), template.Imports("sort")),
	template.WithAfter(fmt.Sprintf(`slices.Sort(%s)`, ssrtS), template.Imports("slices")),
	template.WithCaptures(ssrtS),
)

func (r *PreferSlicesSort) RecipeList() []recipe.Recipe {
	return []recipe.Recipe{preferSlicesSortInts, preferSlicesSortStrings, preferSlicesSortFloat64s}
}
