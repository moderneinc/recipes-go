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

var esS = template.Expr("esS")

// PreferEmptyStringCheck replaces `len(s) == 0` with `s == ""` and
// `len(s) != 0` with `s != ""`.
type PreferEmptyStringCheck struct {
	recipe.Base
}

func (r *PreferEmptyStringCheck) Name() string {
	return "org.openrewrite.golang.codequality.PreferEmptyStringCheck"
}
func (r *PreferEmptyStringCheck) DisplayName() string {
	return "Prefer empty string check"
}
func (r *PreferEmptyStringCheck) Description() string {
	return "Replace `len(s) == 0` with `s == \"\"` and `len(s) != 0` with `s != \"\"`."
}
func (r *PreferEmptyStringCheck) Tags() []string { return []string{"cleanup", "simplification"} }

func (r *PreferEmptyStringCheck) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{}
}

var preferEmptyStringCheckEqual = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.PreferEmptyStringCheck$Equal"),
	template.WithDisplayName("len(s) == 0 → s == \"\""),
	template.WithBefore(fmt.Sprintf(`len(%s) == 0`, esS)),
	template.WithAfter(fmt.Sprintf(`%s == ""`, esS)),
	template.WithCaptures(esS),
)

var preferEmptyStringCheckNotEqual = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.PreferEmptyStringCheck$NotEqual"),
	template.WithDisplayName("len(s) != 0 → s != \"\""),
	template.WithBefore(fmt.Sprintf(`len(%s) != 0`, esS)),
	template.WithAfter(fmt.Sprintf(`%s != ""`, esS)),
	template.WithCaptures(esS),
)

func (r *PreferEmptyStringCheck) RecipeList() []recipe.Recipe {
	return []recipe.Recipe{preferEmptyStringCheckEqual, preferEmptyStringCheckNotEqual}
}
