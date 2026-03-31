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

var oeErr = template.Expr("oeErr")

// PreferErrorsIsForOsCheck replaces `os.IsNotExist(err)` with
// `errors.Is(err, fs.ErrNotExist)` and `os.IsExist(err)` with
// `errors.Is(err, fs.ErrExist)` (Go 1.16+).
// Staticcheck: SA1019 (deprecated)
type PreferErrorsIsForOsCheck struct {
	recipe.Base
}

func (r *PreferErrorsIsForOsCheck) Name() string {
	return "org.openrewrite.golang.codequality.PreferErrorsIsForOsCheck"
}
func (r *PreferErrorsIsForOsCheck) DisplayName() string {
	return "Prefer errors.Is for os existence checks"
}
func (r *PreferErrorsIsForOsCheck) Description() string {
	return "Replace deprecated `os.IsNotExist(err)` with `errors.Is(err, fs.ErrNotExist)` and `os.IsExist(err)` with `errors.Is(err, fs.ErrExist)` (Go 1.16+)."
}
func (r *PreferErrorsIsForOsCheck) Tags() []string { return []string{"cleanup", "simplification"} }

func (r *PreferErrorsIsForOsCheck) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{
		{DiagnosticID: "SA1019", Tool: diagnostic.Staticcheck, HasFix: true},
	}
}

var preferErrorsIsNotExistImpl = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.PreferErrorsIsForOsCheck$NotExist"),
	template.WithDisplayName("os.IsNotExist → errors.Is(err, fs.ErrNotExist)"),
	template.WithBefore(fmt.Sprintf(`os.IsNotExist(%s)`, oeErr), template.Imports("os")),
	template.WithAfter(fmt.Sprintf(`errors.Is(%s, fs.ErrNotExist)`, oeErr), template.Imports("errors", "io/fs")),
	template.WithCaptures(oeErr),
)

var preferErrorsIsExistImpl = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.PreferErrorsIsForOsCheck$Exist"),
	template.WithDisplayName("os.IsExist → errors.Is(err, fs.ErrExist)"),
	template.WithBefore(fmt.Sprintf(`os.IsExist(%s)`, oeErr), template.Imports("os")),
	template.WithAfter(fmt.Sprintf(`errors.Is(%s, fs.ErrExist)`, oeErr), template.Imports("errors", "io/fs")),
	template.WithCaptures(oeErr),
)

func (r *PreferErrorsIsForOsCheck) RecipeList() []recipe.Recipe {
	return []recipe.Recipe{preferErrorsIsNotExistImpl, preferErrorsIsExistImpl}
}
