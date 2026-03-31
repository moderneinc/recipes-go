/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling

import (
	"fmt"

	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/template"
)

var netErr = template.Expr("netErr")

var preferErrorsIsNetClosedEqualImpl = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.PreferErrorsIsNetClosed$Equal"),
	template.WithDisplayName("err == net.ErrClosed -> errors.Is(err, net.ErrClosed)"),
	template.WithBefore(fmt.Sprintf(`%s == net.ErrClosed`, netErr), template.Imports("net")),
	template.WithAfter(fmt.Sprintf(`errors.Is(%s, net.ErrClosed)`, netErr), template.Imports("errors", "net")),
	template.WithCaptures(netErr),
)

var preferErrorsIsNetClosedNotEqualImpl = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.PreferErrorsIsNetClosed$NotEqual"),
	template.WithDisplayName("err != net.ErrClosed -> !errors.Is(err, net.ErrClosed)"),
	template.WithBefore(fmt.Sprintf(`%s != net.ErrClosed`, netErr), template.Imports("net")),
	template.WithAfter(fmt.Sprintf(`!errors.Is(%s, net.ErrClosed)`, netErr), template.Imports("errors", "net")),
	template.WithCaptures(netErr),
)

// PreferErrorsIsNetClosed replaces `err == net.ErrClosed` with
// `errors.Is(err, net.ErrClosed)` and `err != net.ErrClosed` with
// `!errors.Is(err, net.ErrClosed)`. Using errors.Is handles wrapped errors.
type PreferErrorsIsNetClosed struct {
	recipe.Base
}

func (r *PreferErrorsIsNetClosed) Name() string {
	return "org.openrewrite.golang.codequality.PreferErrorsIsNetClosed"
}
func (r *PreferErrorsIsNetClosed) DisplayName() string {
	return "Prefer errors.Is for net.ErrClosed comparison"
}
func (r *PreferErrorsIsNetClosed) Description() string {
	return "Replace `err == net.ErrClosed` with `errors.Is(err, net.ErrClosed)` and `err != net.ErrClosed` with `!errors.Is(err, net.ErrClosed)` for correct wrapped error handling."
}
func (r *PreferErrorsIsNetClosed) Tags() []string { return []string{"error-handling"} }

func (r *PreferErrorsIsNetClosed) RecipeList() []recipe.Recipe {
	return []recipe.Recipe{preferErrorsIsNetClosedEqualImpl, preferErrorsIsNetClosedNotEqualImpl}
}
