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

var otErr = template.Expr("otErr")

// PreferOsIsTimeout replaces `os.IsTimeout(err)` with
// `errors.Is(err, os.ErrDeadlineExceeded)` (Go 1.16+).
// Staticcheck: SA1019 (deprecated)
type PreferOsIsTimeout struct {
	recipe.Base
}

func (r *PreferOsIsTimeout) Name() string {
	return "org.openrewrite.golang.codequality.PreferOsIsTimeout"
}
func (r *PreferOsIsTimeout) DisplayName() string {
	return "Prefer errors.Is for os timeout checks"
}
func (r *PreferOsIsTimeout) Description() string {
	return "Replace deprecated `os.IsTimeout(err)` with `errors.Is(err, os.ErrDeadlineExceeded)` (Go 1.16+)."
}
func (r *PreferOsIsTimeout) Tags() []string { return []string{"cleanup", "simplification"} }

func (r *PreferOsIsTimeout) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{
		{DiagnosticID: "SA1019", Tool: diagnostic.Staticcheck, HasFix: true},
	}
}

var preferOsIsTimeoutImpl = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.PreferOsIsTimeout$Impl"),
	template.WithDisplayName("os.IsTimeout → errors.Is(err, os.ErrDeadlineExceeded)"),
	template.WithBefore(fmt.Sprintf(`os.IsTimeout(%s)`, otErr), template.Imports("os")),
	template.WithAfter(fmt.Sprintf(`errors.Is(%s, os.ErrDeadlineExceeded)`, otErr), template.Imports("errors", "os")),
	template.WithCaptures(otErr),
)

func (r *PreferOsIsTimeout) RecipeList() []recipe.Recipe {
	return []recipe.Recipe{preferOsIsTimeoutImpl}
}
