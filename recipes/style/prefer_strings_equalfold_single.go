/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"fmt"

	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/template"
)

var (
	efsS = template.Expr("efsS")
	efsT = template.Expr("efsT")
)

// PreferStringsEqualFoldSingle replaces `strings.ToLower(s) == t` and
// `strings.ToUpper(s) == t` with `strings.EqualFold(s, t)` for case-insensitive
// string comparison. This handles the single-sided case where only one operand
// uses ToLower/ToUpper.
type PreferStringsEqualFoldSingle struct {
	recipe.Base
}

func (r *PreferStringsEqualFoldSingle) Name() string {
	return "org.openrewrite.golang.codequality.PreferStringsEqualFoldSingle"
}
func (r *PreferStringsEqualFoldSingle) DisplayName() string {
	return "Prefer strings.EqualFold (single-sided)"
}
func (r *PreferStringsEqualFoldSingle) Description() string {
	return "Replace `strings.ToLower(s) == t` and `strings.ToUpper(s) == t` with `strings.EqualFold(s, t)`."
}
func (r *PreferStringsEqualFoldSingle) Tags() []string {
	return []string{"cleanup", "simplification"}
}

// strings.ToLower(s) == t -> strings.EqualFold(s, t)
var preferEqualFoldSingleLowerLeft = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.PreferStringsEqualFoldSingle$LowerLeft"),
	template.WithDisplayName("strings.ToLower(s) == t -> strings.EqualFold(s, t)"),
	template.WithBefore(
		fmt.Sprintf(`strings.ToLower(%s) == %s`, efsS, efsT),
		template.Imports("strings"),
	),
	template.WithAfter(
		fmt.Sprintf(`strings.EqualFold(%s, %s)`, efsS, efsT),
		template.Imports("strings"),
	),
	template.WithCaptures(efsS, efsT),
)

// t == strings.ToLower(s) -> strings.EqualFold(s, t)
var preferEqualFoldSingleLowerRight = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.PreferStringsEqualFoldSingle$LowerRight"),
	template.WithDisplayName("t == strings.ToLower(s) -> strings.EqualFold(s, t)"),
	template.WithBefore(
		fmt.Sprintf(`%s == strings.ToLower(%s)`, efsT, efsS),
		template.Imports("strings"),
	),
	template.WithAfter(
		fmt.Sprintf(`strings.EqualFold(%s, %s)`, efsS, efsT),
		template.Imports("strings"),
	),
	template.WithCaptures(efsS, efsT),
)

// strings.ToUpper(s) == t -> strings.EqualFold(s, t)
var preferEqualFoldSingleUpperLeft = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.PreferStringsEqualFoldSingle$UpperLeft"),
	template.WithDisplayName("strings.ToUpper(s) == t -> strings.EqualFold(s, t)"),
	template.WithBefore(
		fmt.Sprintf(`strings.ToUpper(%s) == %s`, efsS, efsT),
		template.Imports("strings"),
	),
	template.WithAfter(
		fmt.Sprintf(`strings.EqualFold(%s, %s)`, efsS, efsT),
		template.Imports("strings"),
	),
	template.WithCaptures(efsS, efsT),
)

// t == strings.ToUpper(s) -> strings.EqualFold(s, t)
var preferEqualFoldSingleUpperRight = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.PreferStringsEqualFoldSingle$UpperRight"),
	template.WithDisplayName("t == strings.ToUpper(s) -> strings.EqualFold(s, t)"),
	template.WithBefore(
		fmt.Sprintf(`%s == strings.ToUpper(%s)`, efsT, efsS),
		template.Imports("strings"),
	),
	template.WithAfter(
		fmt.Sprintf(`strings.EqualFold(%s, %s)`, efsS, efsT),
		template.Imports("strings"),
	),
	template.WithCaptures(efsS, efsT),
)

func (r *PreferStringsEqualFoldSingle) RecipeList() []recipe.Recipe {
	return []recipe.Recipe{
		preferEqualFoldSingleLowerLeft,
		preferEqualFoldSingleLowerRight,
		preferEqualFoldSingleUpperLeft,
		preferEqualFoldSingleUpperRight,
	}
}
