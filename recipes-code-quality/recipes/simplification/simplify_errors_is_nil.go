/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification

import (
	"fmt"

	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/template"
)

var einErr = template.Expr("einErr")

// SimplifyErrorsIsNil replaces `errors.Is(err, nil)` with `err == nil`.
// The errors.Is call with a nil target is redundant since nil does not wrap.
type SimplifyErrorsIsNil struct {
	recipe.Base
}

func (r *SimplifyErrorsIsNil) Name() string {
	return "org.openrewrite.golang.codequality.SimplifyErrorsIsNil"
}
func (r *SimplifyErrorsIsNil) DisplayName() string {
	return "Simplify errors.Is nil check"
}
func (r *SimplifyErrorsIsNil) Description() string {
	return "Replace redundant `errors.Is(err, nil)` with `err == nil`."
}
func (r *SimplifyErrorsIsNil) Tags() []string { return []string{"cleanup", "simplification"} }

var simplifyErrorsIsNilImpl = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.SimplifyErrorsIsNil$Impl"),
	template.WithDisplayName("errors.Is(err, nil) → err == nil"),
	template.WithBefore(fmt.Sprintf(`errors.Is(%s, nil)`, einErr), template.Imports("errors")),
	template.WithAfter(fmt.Sprintf(`%s == nil`, einErr)),
	template.WithCaptures(einErr),
)

func (r *SimplifyErrorsIsNil) RecipeList() []recipe.Recipe {
	return []recipe.Recipe{simplifyErrorsIsNilImpl}
}
