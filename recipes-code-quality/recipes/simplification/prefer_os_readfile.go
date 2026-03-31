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

var rfName = template.Expr("rfName")

// PreferOsReadFile replaces `ioutil.ReadFile(name)` with `os.ReadFile(name)` (Go 1.16+).
// Staticcheck: SA1019 (deprecated)
type PreferOsReadFile struct {
	recipe.Base
}

func (r *PreferOsReadFile) Name() string {
	return "org.openrewrite.golang.codequality.PreferOsReadFile"
}
func (r *PreferOsReadFile) DisplayName() string {
	return "Prefer os.ReadFile"
}
func (r *PreferOsReadFile) Description() string {
	return "Replace deprecated `ioutil.ReadFile(name)` with `os.ReadFile(name)` (Go 1.16+)."
}
func (r *PreferOsReadFile) Tags() []string { return []string{"cleanup", "simplification"} }

func (r *PreferOsReadFile) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{
		{DiagnosticID: "SA1019", Tool: diagnostic.Staticcheck, HasFix: true},
	}
}

var preferOsReadFileImpl = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.PreferOsReadFile$Impl"),
	template.WithDisplayName("ioutil.ReadFile → os.ReadFile"),
	template.WithBefore(fmt.Sprintf(`ioutil.ReadFile(%s)`, rfName), template.Imports("io/ioutil")),
	template.WithAfter(fmt.Sprintf(`os.ReadFile(%s)`, rfName), template.Imports("os")),
	template.WithCaptures(rfName),
)

func (r *PreferOsReadFile) RecipeList() []recipe.Recipe {
	return []recipe.Recipe{preferOsReadFileImpl}
}
