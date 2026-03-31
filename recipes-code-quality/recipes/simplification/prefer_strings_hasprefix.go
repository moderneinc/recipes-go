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
	hpS      = template.Expr("s")
	hpPrefix = template.Expr("prefix")
)

// PreferStringsHasPrefix replaces `strings.Index(s, prefix) == 0` with
// `strings.HasPrefix(s, prefix)`. Staticcheck: S1003
type PreferStringsHasPrefix struct {
	recipe.Base
}

func (r *PreferStringsHasPrefix) Name() string {
	return "org.openrewrite.golang.codequality.PreferStringsHasPrefix"
}
func (r *PreferStringsHasPrefix) DisplayName() string { return "Prefer strings.HasPrefix" }
func (r *PreferStringsHasPrefix) Description() string {
	return "Replace `strings.Index(s, prefix) == 0` with `strings.HasPrefix(s, prefix)`."
}
func (r *PreferStringsHasPrefix) Tags() []string { return []string{"cleanup", "simplification"} }

func (r *PreferStringsHasPrefix) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{
		{DiagnosticID: "S1003", Tool: diagnostic.Staticcheck, HasFix: true},
	}
}

var preferStringsHasPrefixPositive = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.PreferStringsHasPrefix$Positive"),
	template.WithDisplayName("strings.Index == 0 → strings.HasPrefix"),
	template.WithBefore(fmt.Sprintf(`strings.Index(%s, %s) == 0`, hpS, hpPrefix), template.Imports("strings")),
	template.WithAfter(fmt.Sprintf(`strings.HasPrefix(%s, %s)`, hpS, hpPrefix), template.Imports("strings")),
	template.WithCaptures(hpS, hpPrefix),
)

var preferStringsHasPrefixNegative = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.PreferStringsHasPrefix$Negative"),
	template.WithDisplayName("strings.Index != 0 → !strings.HasPrefix"),
	template.WithBefore(fmt.Sprintf(`strings.Index(%s, %s) != 0`, hpS, hpPrefix), template.Imports("strings")),
	template.WithAfter(fmt.Sprintf(`!strings.HasPrefix(%s, %s)`, hpS, hpPrefix), template.Imports("strings")),
	template.WithCaptures(hpS, hpPrefix),
)

func (r *PreferStringsHasPrefix) RecipeList() []recipe.Recipe {
	return []recipe.Recipe{preferStringsHasPrefixPositive, preferStringsHasPrefixNegative}
}
