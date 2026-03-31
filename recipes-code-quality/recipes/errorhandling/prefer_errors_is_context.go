/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling

import (
	"fmt"

	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/template"
)

var ctxErr = template.Expr("ctxErr")

var preferErrorsIsContextCanceledEqualImpl = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.PreferErrorsIsContext$CanceledEqual"),
	template.WithDisplayName("err == context.Canceled -> errors.Is(err, context.Canceled)"),
	template.WithBefore(fmt.Sprintf(`%s == context.Canceled`, ctxErr), template.Imports("context")),
	template.WithAfter(fmt.Sprintf(`errors.Is(%s, context.Canceled)`, ctxErr), template.Imports("errors", "context")),
	template.WithCaptures(ctxErr),
)

var preferErrorsIsContextCanceledNotEqualImpl = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.PreferErrorsIsContext$CanceledNotEqual"),
	template.WithDisplayName("err != context.Canceled -> !errors.Is(err, context.Canceled)"),
	template.WithBefore(fmt.Sprintf(`%s != context.Canceled`, ctxErr), template.Imports("context")),
	template.WithAfter(fmt.Sprintf(`!errors.Is(%s, context.Canceled)`, ctxErr), template.Imports("errors", "context")),
	template.WithCaptures(ctxErr),
)

var preferErrorsIsContextDeadlineEqualImpl = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.PreferErrorsIsContext$DeadlineEqual"),
	template.WithDisplayName("err == context.DeadlineExceeded -> errors.Is(err, context.DeadlineExceeded)"),
	template.WithBefore(fmt.Sprintf(`%s == context.DeadlineExceeded`, ctxErr), template.Imports("context")),
	template.WithAfter(fmt.Sprintf(`errors.Is(%s, context.DeadlineExceeded)`, ctxErr), template.Imports("errors", "context")),
	template.WithCaptures(ctxErr),
)

var preferErrorsIsContextDeadlineNotEqualImpl = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.PreferErrorsIsContext$DeadlineNotEqual"),
	template.WithDisplayName("err != context.DeadlineExceeded -> !errors.Is(err, context.DeadlineExceeded)"),
	template.WithBefore(fmt.Sprintf(`%s != context.DeadlineExceeded`, ctxErr), template.Imports("context")),
	template.WithAfter(fmt.Sprintf(`!errors.Is(%s, context.DeadlineExceeded)`, ctxErr), template.Imports("errors", "context")),
	template.WithCaptures(ctxErr),
)

// PreferErrorsIsContext replaces `err == context.Canceled` with
// `errors.Is(err, context.Canceled)` and `err == context.DeadlineExceeded` with
// `errors.Is(err, context.DeadlineExceeded)`, plus their != variants.
// Using errors.Is handles wrapped errors correctly.
type PreferErrorsIsContext struct {
	recipe.Base
}

func (r *PreferErrorsIsContext) Name() string {
	return "org.openrewrite.golang.codequality.PreferErrorsIsContext"
}
func (r *PreferErrorsIsContext) DisplayName() string {
	return "Prefer errors.Is for context error comparison"
}
func (r *PreferErrorsIsContext) Description() string {
	return "Replace `err == context.Canceled` and `err == context.DeadlineExceeded` with `errors.Is` for correct wrapped error handling."
}
func (r *PreferErrorsIsContext) Tags() []string { return []string{"error-handling"} }

func (r *PreferErrorsIsContext) RecipeList() []recipe.Recipe {
	return []recipe.Recipe{
		preferErrorsIsContextCanceledEqualImpl,
		preferErrorsIsContextCanceledNotEqualImpl,
		preferErrorsIsContextDeadlineEqualImpl,
		preferErrorsIsContextDeadlineNotEqualImpl,
	}
}
