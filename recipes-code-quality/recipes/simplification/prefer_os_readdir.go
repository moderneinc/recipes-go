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

var rdirName = template.Expr("rdirName")

// PreferOsReadDir replaces `ioutil.ReadDir(name)` with `os.ReadDir(name)` (Go 1.16+).
// Staticcheck: SA1019 (deprecated)
type PreferOsReadDir struct {
	recipe.Base
}

func (r *PreferOsReadDir) Name() string {
	return "org.openrewrite.golang.codequality.PreferOsReadDir"
}
func (r *PreferOsReadDir) DisplayName() string {
	return "Prefer os.ReadDir"
}
func (r *PreferOsReadDir) Description() string {
	return "Replace deprecated `ioutil.ReadDir(name)` with `os.ReadDir(name)` (Go 1.16+)."
}
func (r *PreferOsReadDir) Tags() []string { return []string{"cleanup", "simplification"} }

func (r *PreferOsReadDir) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{
		{DiagnosticID: "SA1019", Tool: diagnostic.Staticcheck, HasFix: true},
	}
}

var preferOsReadDirImpl = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.PreferOsReadDir$Impl"),
	template.WithDisplayName("ioutil.ReadDir \u2192 os.ReadDir"),
	template.WithBefore(fmt.Sprintf(`ioutil.ReadDir(%s)`, rdirName), template.Imports("io/ioutil")),
	template.WithAfter(fmt.Sprintf(`os.ReadDir(%s)`, rdirName), template.Imports("os")),
	template.WithCaptures(rdirName),
)

func (r *PreferOsReadDir) RecipeList() []recipe.Recipe {
	return []recipe.Recipe{preferOsReadDirImpl}
}
