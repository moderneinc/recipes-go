/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling

import (
	"fmt"

	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/template"
)

var opiErr = template.Expr("opiErr")

var preferErrorsIsOsInvalidEqualImpl = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.PreferErrorsIsOsInvalid$Equal"),
	template.WithDisplayName("err == os.ErrInvalid -> errors.Is(err, os.ErrInvalid)"),
	template.WithBefore(fmt.Sprintf(`%s == os.ErrInvalid`, opiErr), template.Imports("os")),
	template.WithAfter(fmt.Sprintf(`errors.Is(%s, os.ErrInvalid)`, opiErr), template.Imports("errors", "os")),
	template.WithCaptures(opiErr),
)

// PreferErrorsIsOsInvalid replaces `err == os.ErrInvalid` with
// `errors.Is(err, os.ErrInvalid)`. Using errors.Is handles wrapped errors.
type PreferErrorsIsOsInvalid struct {
	recipe.Base
}

func (r *PreferErrorsIsOsInvalid) Name() string {
	return "org.openrewrite.golang.codequality.PreferErrorsIsOsInvalid"
}
func (r *PreferErrorsIsOsInvalid) DisplayName() string {
	return "Prefer errors.Is for os.ErrInvalid comparison"
}
func (r *PreferErrorsIsOsInvalid) Description() string {
	return "Replace `err == os.ErrInvalid` with `errors.Is(err, os.ErrInvalid)` for correct wrapped error handling."
}
func (r *PreferErrorsIsOsInvalid) Tags() []string { return []string{"error-handling"} }

func (r *PreferErrorsIsOsInvalid) RecipeList() []recipe.Recipe {
	return []recipe.Recipe{preferErrorsIsOsInvalidEqualImpl}
}
