/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling

import (
	"fmt"

	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/template"
)

var htErr = template.Expr("htErr")

var preferErrorsIsHttpServerClosedEqualImpl = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.PreferErrorsIsHttpServerClosed$Equal"),
	template.WithDisplayName("err == http.ErrServerClosed -> errors.Is(err, http.ErrServerClosed)"),
	template.WithBefore(fmt.Sprintf(`%s == http.ErrServerClosed`, htErr), template.Imports("net/http")),
	template.WithAfter(fmt.Sprintf(`errors.Is(%s, http.ErrServerClosed)`, htErr), template.Imports("errors", "net/http")),
	template.WithCaptures(htErr),
)

var preferErrorsIsHttpServerClosedNotEqualImpl = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.PreferErrorsIsHttpServerClosed$NotEqual"),
	template.WithDisplayName("err != http.ErrServerClosed -> !errors.Is(err, http.ErrServerClosed)"),
	template.WithBefore(fmt.Sprintf(`%s != http.ErrServerClosed`, htErr), template.Imports("net/http")),
	template.WithAfter(fmt.Sprintf(`!errors.Is(%s, http.ErrServerClosed)`, htErr), template.Imports("errors", "net/http")),
	template.WithCaptures(htErr),
)

// PreferErrorsIsHttpServerClosed replaces `err == http.ErrServerClosed` with
// `errors.Is(err, http.ErrServerClosed)` and `err != http.ErrServerClosed` with
// `!errors.Is(err, http.ErrServerClosed)`. Using errors.Is handles wrapped errors.
type PreferErrorsIsHttpServerClosed struct {
	recipe.Base
}

func (r *PreferErrorsIsHttpServerClosed) Name() string {
	return "org.openrewrite.golang.codequality.PreferErrorsIsHttpServerClosed"
}
func (r *PreferErrorsIsHttpServerClosed) DisplayName() string {
	return "Prefer errors.Is for http.ErrServerClosed comparison"
}
func (r *PreferErrorsIsHttpServerClosed) Description() string {
	return "Replace `err == http.ErrServerClosed` with `errors.Is(err, http.ErrServerClosed)` and `err != http.ErrServerClosed` with `!errors.Is(err, http.ErrServerClosed)` for correct wrapped error handling."
}
func (r *PreferErrorsIsHttpServerClosed) Tags() []string { return []string{"error-handling"} }

func (r *PreferErrorsIsHttpServerClosed) RecipeList() []recipe.Recipe {
	return []recipe.Recipe{preferErrorsIsHttpServerClosedEqualImpl, preferErrorsIsHttpServerClosedNotEqualImpl}
}
