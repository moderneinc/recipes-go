/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification

import (
	"fmt"

	"github.com/moderneinc/recipes-go/code-quality/diagnostic"
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/template"
)

var (
	scmpA = template.Expr("scmpA")
	scmpB = template.Expr("scmpB")
)

// PreferStringComparison replaces `strings.Compare(a, b)` comparison patterns
// with direct string comparison operators.
// Staticcheck: S1021
type PreferStringComparison struct {
	recipe.Base
}

func (r *PreferStringComparison) Name() string {
	return "org.openrewrite.golang.codequality.PreferStringComparison"
}
func (r *PreferStringComparison) DisplayName() string {
	return "Prefer string comparison operators"
}
func (r *PreferStringComparison) Description() string {
	return "Replace `strings.Compare(a, b) == 0` with `a == b`, `strings.Compare(a, b) != 0` with `a != b`, `strings.Compare(a, b) < 0` with `a < b`, and `strings.Compare(a, b) > 0` with `a > b`."
}
func (r *PreferStringComparison) Tags() []string { return []string{"cleanup", "simplification"} }

func (r *PreferStringComparison) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{
		{DiagnosticID: "S1021", Tool: diagnostic.Staticcheck, HasFix: true},
	}
}

var preferStringComparisonEq = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.PreferStringComparison$Eq"),
	template.WithDisplayName("strings.Compare == 0 -> =="),
	template.WithBefore(fmt.Sprintf(`strings.Compare(%s, %s) == 0`, scmpA, scmpB), template.Imports("strings")),
	template.WithAfter(fmt.Sprintf(`%s == %s`, scmpA, scmpB)),
	template.WithCaptures(scmpA, scmpB),
)

var preferStringComparisonNeq = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.PreferStringComparison$Neq"),
	template.WithDisplayName("strings.Compare != 0 -> !="),
	template.WithBefore(fmt.Sprintf(`strings.Compare(%s, %s) != 0`, scmpA, scmpB), template.Imports("strings")),
	template.WithAfter(fmt.Sprintf(`%s != %s`, scmpA, scmpB)),
	template.WithCaptures(scmpA, scmpB),
)

var preferStringComparisonLt = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.PreferStringComparison$Lt"),
	template.WithDisplayName("strings.Compare < 0 -> <"),
	template.WithBefore(fmt.Sprintf(`strings.Compare(%s, %s) < 0`, scmpA, scmpB), template.Imports("strings")),
	template.WithAfter(fmt.Sprintf(`%s < %s`, scmpA, scmpB)),
	template.WithCaptures(scmpA, scmpB),
)

var preferStringComparisonGt = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.PreferStringComparison$Gt"),
	template.WithDisplayName("strings.Compare > 0 -> >"),
	template.WithBefore(fmt.Sprintf(`strings.Compare(%s, %s) > 0`, scmpA, scmpB), template.Imports("strings")),
	template.WithAfter(fmt.Sprintf(`%s > %s`, scmpA, scmpB)),
	template.WithCaptures(scmpA, scmpB),
)

func (r *PreferStringComparison) RecipeList() []recipe.Recipe {
	return []recipe.Recipe{
		preferStringComparisonEq,
		preferStringComparisonNeq,
		preferStringComparisonLt,
		preferStringComparisonGt,
	}
}
