/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification

import (
	"fmt"

	"github.com/moderneinc/recipes-go/recipes-code-quality/diagnostic"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/template"
)

var (
	sortS = template.Expr("s")
)

// PreferSortInts replaces `sort.Slice(s, func(i, j int) bool { return s[i] < s[j] })`
// patterns. Note: the full pattern is too complex for template matching, so this
// targets the simpler case of `sort.Sort(sort.IntSlice(s))` → `sort.Ints(s)`.
// Staticcheck: S1032
type PreferSortInts struct {
	recipe.Base
}

func (r *PreferSortInts) Name() string {
	return "org.openrewrite.golang.codequality.PreferSortInts"
}
func (r *PreferSortInts) DisplayName() string { return "Prefer sort.Ints over sort.Sort(sort.IntSlice)" }
func (r *PreferSortInts) Description() string {
	return "Replace `sort.Sort(sort.IntSlice(s))` with `sort.Ints(s)`."
}
func (r *PreferSortInts) Tags() []string { return []string{"cleanup", "simplification"} }

func (r *PreferSortInts) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{
		{DiagnosticID: "S1032", Tool: diagnostic.Staticcheck, HasFix: true},
	}
}

var preferSortIntsImpl = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.PreferSortInts$Impl"),
	template.WithDisplayName("sort.Sort(sort.IntSlice) → sort.Ints"),
	template.WithBefore(fmt.Sprintf(`sort.Sort(sort.IntSlice(%s))`, sortS), template.Imports("sort")),
	template.WithAfter(fmt.Sprintf(`sort.Ints(%s)`, sortS), template.Imports("sort")),
	template.WithCaptures(sortS),
)

var preferSortStringsImpl = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.PreferSortInts$Strings"),
	template.WithDisplayName("sort.Sort(sort.StringSlice) → sort.Strings"),
	template.WithBefore(fmt.Sprintf(`sort.Sort(sort.StringSlice(%s))`, sortS), template.Imports("sort")),
	template.WithAfter(fmt.Sprintf(`sort.Strings(%s)`, sortS), template.Imports("sort")),
	template.WithCaptures(sortS),
)

var preferSortFloat64sImpl = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.PreferSortInts$Float64s"),
	template.WithDisplayName("sort.Sort(sort.Float64Slice) → sort.Float64s"),
	template.WithBefore(fmt.Sprintf(`sort.Sort(sort.Float64Slice(%s))`, sortS), template.Imports("sort")),
	template.WithAfter(fmt.Sprintf(`sort.Float64s(%s)`, sortS), template.Imports("sort")),
	template.WithCaptures(sortS),
)

func (r *PreferSortInts) RecipeList() []recipe.Recipe {
	return []recipe.Recipe{preferSortIntsImpl, preferSortStringsImpl, preferSortFloat64sImpl}
}
