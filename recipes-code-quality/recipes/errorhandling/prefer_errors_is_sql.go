/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling

import (
	"fmt"

	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/template"
)

var sqlErr = template.Expr("sqlErr")

var preferErrorsIsSqlNoRowsEqualImpl = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.PreferErrorsIsSqlNoRows$Equal"),
	template.WithDisplayName("err == sql.ErrNoRows -> errors.Is(err, sql.ErrNoRows)"),
	template.WithBefore(fmt.Sprintf(`%s == sql.ErrNoRows`, sqlErr), template.Imports("database/sql")),
	template.WithAfter(fmt.Sprintf(`errors.Is(%s, sql.ErrNoRows)`, sqlErr), template.Imports("errors", "database/sql")),
	template.WithCaptures(sqlErr),
)

var preferErrorsIsSqlNoRowsNotEqualImpl = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.PreferErrorsIsSqlNoRows$NotEqual"),
	template.WithDisplayName("err != sql.ErrNoRows -> !errors.Is(err, sql.ErrNoRows)"),
	template.WithBefore(fmt.Sprintf(`%s != sql.ErrNoRows`, sqlErr), template.Imports("database/sql")),
	template.WithAfter(fmt.Sprintf(`!errors.Is(%s, sql.ErrNoRows)`, sqlErr), template.Imports("errors", "database/sql")),
	template.WithCaptures(sqlErr),
)

// PreferErrorsIsSqlNoRows replaces `err == sql.ErrNoRows` with `errors.Is(err, sql.ErrNoRows)` and
// `err != sql.ErrNoRows` with `!errors.Is(err, sql.ErrNoRows)`. The sql.ErrNoRows sentinel is a
// common error value compared by ==; using errors.Is handles wrapped errors.
type PreferErrorsIsSqlNoRows struct {
	recipe.Base
}

func (r *PreferErrorsIsSqlNoRows) Name() string {
	return "org.openrewrite.golang.codequality.PreferErrorsIsSqlNoRows"
}
func (r *PreferErrorsIsSqlNoRows) DisplayName() string {
	return "Prefer errors.Is for sql.ErrNoRows comparison"
}
func (r *PreferErrorsIsSqlNoRows) Description() string {
	return "Replace `err == sql.ErrNoRows` with `errors.Is(err, sql.ErrNoRows)` and `err != sql.ErrNoRows` with `!errors.Is(err, sql.ErrNoRows)` for correct wrapped error handling."
}
func (r *PreferErrorsIsSqlNoRows) Tags() []string { return []string{"error-handling"} }

func (r *PreferErrorsIsSqlNoRows) RecipeList() []recipe.Recipe {
	return []recipe.Recipe{preferErrorsIsSqlNoRowsEqualImpl, preferErrorsIsSqlNoRowsNotEqualImpl}
}
