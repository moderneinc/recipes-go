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

var stlS = template.Expr("stlS")

// PreferStringsToLowerMap replaces `strings.Map(unicode.ToLower, s)` with
// `strings.ToLower(s)`. Staticcheck: S1029
type PreferStringsToLowerMap struct {
	recipe.Base
}

func (r *PreferStringsToLowerMap) Name() string {
	return "org.openrewrite.golang.codequality.PreferStringsToLowerMap"
}
func (r *PreferStringsToLowerMap) DisplayName() string {
	return "Prefer strings.ToLower over strings.Map"
}
func (r *PreferStringsToLowerMap) Description() string {
	return "Replace `strings.Map(unicode.ToLower, s)` with `strings.ToLower(s)`."
}
func (r *PreferStringsToLowerMap) Tags() []string { return []string{"cleanup", "simplification"} }

func (r *PreferStringsToLowerMap) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{
		{DiagnosticID: "S1029", Tool: diagnostic.Staticcheck, HasFix: true},
	}
}

var preferStringsToLowerMapImpl = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.PreferStringsToLowerMap$Impl"),
	template.WithDisplayName("strings.Map(unicode.ToLower, s) -> strings.ToLower(s)"),
	template.WithBefore(fmt.Sprintf(`strings.Map(unicode.ToLower, %s)`, stlS), template.Imports("strings", "unicode")),
	template.WithAfter(fmt.Sprintf(`strings.ToLower(%s)`, stlS), template.Imports("strings")),
	template.WithCaptures(stlS),
)

func (r *PreferStringsToLowerMap) RecipeList() []recipe.Recipe {
	return []recipe.Recipe{preferStringsToLowerMapImpl}
}
