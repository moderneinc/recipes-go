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
	tfDir = template.Expr("tfDir")
	tfPat = template.Expr("tfPat")
)

// PreferOsCreateTemp replaces `ioutil.TempFile(dir, pattern)` with
// `os.CreateTemp(dir, pattern)` (Go 1.16+).
// Staticcheck: SA1019 (deprecated)
type PreferOsCreateTemp struct {
	recipe.Base
}

func (r *PreferOsCreateTemp) Name() string {
	return "org.openrewrite.golang.codequality.PreferOsCreateTemp"
}
func (r *PreferOsCreateTemp) DisplayName() string {
	return "Prefer os.CreateTemp"
}
func (r *PreferOsCreateTemp) Description() string {
	return "Replace deprecated `ioutil.TempFile(dir, pattern)` with `os.CreateTemp(dir, pattern)` (Go 1.16+)."
}
func (r *PreferOsCreateTemp) Tags() []string { return []string{"cleanup", "simplification"} }

func (r *PreferOsCreateTemp) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{
		{DiagnosticID: "SA1019", Tool: diagnostic.Staticcheck, HasFix: true},
	}
}

var preferOsCreateTempImpl = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.PreferOsCreateTemp$Impl"),
	template.WithDisplayName("ioutil.TempFile → os.CreateTemp"),
	template.WithBefore(fmt.Sprintf(`ioutil.TempFile(%s, %s)`, tfDir, tfPat), template.Imports("io/ioutil")),
	template.WithAfter(fmt.Sprintf(`os.CreateTemp(%s, %s)`, tfDir, tfPat), template.Imports("os")),
	template.WithCaptures(tfDir, tfPat),
)

func (r *PreferOsCreateTemp) RecipeList() []recipe.Recipe {
	return []recipe.Recipe{preferOsCreateTempImpl}
}
