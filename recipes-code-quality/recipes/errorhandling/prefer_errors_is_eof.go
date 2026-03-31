/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling

import (
	"fmt"

	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/template"
)

var eofErr = template.Expr("eofErr")

var preferErrorsIsEOFEqualImpl = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.PreferErrorsIsEOF$Equal"),
	template.WithDisplayName("err == io.EOF -> errors.Is(err, io.EOF)"),
	template.WithBefore(fmt.Sprintf(`%s == io.EOF`, eofErr), template.Imports("io")),
	template.WithAfter(fmt.Sprintf(`errors.Is(%s, io.EOF)`, eofErr), template.Imports("errors", "io")),
	template.WithCaptures(eofErr),
)

var preferErrorsIsEOFNotEqualImpl = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.PreferErrorsIsEOF$NotEqual"),
	template.WithDisplayName("err != io.EOF -> !errors.Is(err, io.EOF)"),
	template.WithBefore(fmt.Sprintf(`%s != io.EOF`, eofErr), template.Imports("io")),
	template.WithAfter(fmt.Sprintf(`!errors.Is(%s, io.EOF)`, eofErr), template.Imports("errors", "io")),
	template.WithCaptures(eofErr),
)

// PreferErrorsIsEOF replaces `err == io.EOF` with `errors.Is(err, io.EOF)` and
// `err != io.EOF` with `!errors.Is(err, io.EOF)`. The io.EOF sentinel is the
// most common error value compared by ==; using errors.Is handles wrapped errors.
type PreferErrorsIsEOF struct {
	recipe.Base
}

func (r *PreferErrorsIsEOF) Name() string {
	return "org.openrewrite.golang.codequality.PreferErrorsIsEOF"
}
func (r *PreferErrorsIsEOF) DisplayName() string {
	return "Prefer errors.Is for io.EOF comparison"
}
func (r *PreferErrorsIsEOF) Description() string {
	return "Replace `err == io.EOF` with `errors.Is(err, io.EOF)` and `err != io.EOF` with `!errors.Is(err, io.EOF)` for correct wrapped error handling."
}
func (r *PreferErrorsIsEOF) Tags() []string { return []string{"error-handling"} }

func (r *PreferErrorsIsEOF) RecipeList() []recipe.Recipe {
	return []recipe.Recipe{preferErrorsIsEOFEqualImpl, preferErrorsIsEOFNotEqualImpl}
}
