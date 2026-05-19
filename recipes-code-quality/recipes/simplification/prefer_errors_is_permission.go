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

var opErr = template.Expr("opErr")

// PreferErrorsIsForPermission replaces `os.IsPermission(err)` with
// `errors.Is(err, fs.ErrPermission)` (Go 1.16+).
// Staticcheck: SA1019 (deprecated)
type PreferErrorsIsForPermission struct {
	recipe.Base
}

func (r *PreferErrorsIsForPermission) Name() string {
	return "org.openrewrite.golang.codequality.PreferErrorsIsForPermission"
}
func (r *PreferErrorsIsForPermission) DisplayName() string {
	return "Prefer errors.Is for os permission checks"
}
func (r *PreferErrorsIsForPermission) Description() string {
	return "Replace deprecated `os.IsPermission(err)` with `errors.Is(err, fs.ErrPermission)` (Go 1.16+)."
}
func (r *PreferErrorsIsForPermission) Tags() []string { return []string{"cleanup", "simplification"} }

func (r *PreferErrorsIsForPermission) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{
		{DiagnosticID: "SA1019", Tool: diagnostic.Staticcheck, HasFix: true},
	}
}

var preferErrorsIsPermissionImpl = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.PreferErrorsIsForPermission$Impl"),
	template.WithDisplayName("os.IsPermission → errors.Is(err, fs.ErrPermission)"),
	template.WithBefore(fmt.Sprintf(`os.IsPermission(%s)`, opErr), template.Imports("os")),
	template.WithAfter(fmt.Sprintf(`errors.Is(%s, fs.ErrPermission)`, opErr), template.Imports("errors", "io/fs")),
	template.WithCaptures(opErr),
)

func (r *PreferErrorsIsForPermission) RecipeList() []recipe.Recipe {
	return []recipe.Recipe{preferErrorsIsPermissionImpl}
}
