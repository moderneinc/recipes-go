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

var stuS = template.Expr("stuS")

// PreferStringsToUpperMap replaces `strings.Map(unicode.ToUpper, s)` with
// `strings.ToUpper(s)`. Staticcheck: S1029
type PreferStringsToUpperMap struct {
	recipe.Base
}

func (r *PreferStringsToUpperMap) Name() string {
	return "org.openrewrite.golang.codequality.PreferStringsToUpperMap"
}
func (r *PreferStringsToUpperMap) DisplayName() string {
	return "Prefer strings.ToUpper over strings.Map"
}
func (r *PreferStringsToUpperMap) Description() string {
	return "Replace `strings.Map(unicode.ToUpper, s)` with `strings.ToUpper(s)`."
}
func (r *PreferStringsToUpperMap) Tags() []string { return []string{"cleanup", "simplification"} }

func (r *PreferStringsToUpperMap) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{
		{DiagnosticID: "S1029", Tool: diagnostic.Staticcheck, HasFix: true},
	}
}

var preferStringsToUpperMapImpl = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.PreferStringsToUpperMap$Impl"),
	template.WithDisplayName("strings.Map(unicode.ToUpper, s) -> strings.ToUpper(s)"),
	template.WithBefore(fmt.Sprintf(`strings.Map(unicode.ToUpper, %s)`, stuS), template.Imports("strings", "unicode")),
	template.WithAfter(fmt.Sprintf(`strings.ToUpper(%s)`, stuS), template.Imports("strings")),
	template.WithCaptures(stuS),
)

func (r *PreferStringsToUpperMap) RecipeList() []recipe.Recipe {
	return []recipe.Recipe{preferStringsToUpperMapImpl}
}
