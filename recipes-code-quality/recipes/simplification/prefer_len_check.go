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

var lcS = template.Expr("lcS")

// PreferLenCheck normalizes len checks to canonical form:
// `len(s) >= 1` → `len(s) > 0` and `len(s) < 1` → `len(s) == 0`.
type PreferLenCheck struct {
	recipe.Base
}

func (r *PreferLenCheck) Name() string {
	return "org.openrewrite.golang.codequality.PreferLenCheck"
}
func (r *PreferLenCheck) DisplayName() string {
	return "Prefer canonical len check"
}
func (r *PreferLenCheck) Description() string {
	return "Normalize `len(s) >= 1` to `len(s) > 0` and `len(s) < 1` to `len(s) == 0`."
}
func (r *PreferLenCheck) Tags() []string { return []string{"cleanup", "simplification"} }

func (r *PreferLenCheck) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{}
}

var preferLenCheckGte = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.PreferLenCheck$Gte"),
	template.WithDisplayName("len(s) >= 1 → len(s) > 0"),
	template.WithBefore(fmt.Sprintf(`len(%s) >= 1`, lcS)),
	template.WithAfter(fmt.Sprintf(`len(%s) > 0`, lcS)),
	template.WithCaptures(lcS),
)

var preferLenCheckLt = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.PreferLenCheck$Lt"),
	template.WithDisplayName("len(s) < 1 → len(s) == 0"),
	template.WithBefore(fmt.Sprintf(`len(%s) < 1`, lcS)),
	template.WithAfter(fmt.Sprintf(`len(%s) == 0`, lcS)),
	template.WithCaptures(lcS),
)

func (r *PreferLenCheck) RecipeList() []recipe.Recipe {
	return []recipe.Recipe{preferLenCheckGte, preferLenCheckLt}
}
