/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification

import (
	"fmt"

	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/template"
)

var (
	mmxA = template.Expr("mmxA")
	mmxB = template.Expr("mmxB")
)

// PreferMinMaxBuiltin replaces `math.Min(a, b)` with `min(a, b)` and
// `math.Max(a, b)` with `max(a, b)` using the Go 1.21 built-in functions.
type PreferMinMaxBuiltin struct {
	recipe.Base
}

func (r *PreferMinMaxBuiltin) Name() string {
	return "org.openrewrite.golang.codequality.PreferMinMaxBuiltin"
}
func (r *PreferMinMaxBuiltin) DisplayName() string {
	return "Prefer min/max builtins"
}
func (r *PreferMinMaxBuiltin) Description() string {
	return "Replace `math.Min(a, b)` with `min(a, b)` and `math.Max(a, b)` with `max(a, b)` (Go 1.21+)."
}
func (r *PreferMinMaxBuiltin) Tags() []string {
	return []string{"cleanup", "simplification"}
}

var preferMinBuiltin = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.PreferMinMaxBuiltin$Min"),
	template.WithDisplayName("math.Min -> min"),
	template.WithBefore(fmt.Sprintf(`math.Min(%s, %s)`, mmxA, mmxB), template.Imports("math")),
	template.WithAfter(fmt.Sprintf(`min(%s, %s)`, mmxA, mmxB)),
	template.WithCaptures(mmxA, mmxB),
)

var preferMaxBuiltin = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.PreferMinMaxBuiltin$Max"),
	template.WithDisplayName("math.Max -> max"),
	template.WithBefore(fmt.Sprintf(`math.Max(%s, %s)`, mmxA, mmxB), template.Imports("math")),
	template.WithAfter(fmt.Sprintf(`max(%s, %s)`, mmxA, mmxB)),
	template.WithCaptures(mmxA, mmxB),
)

func (r *PreferMinMaxBuiltin) RecipeList() []recipe.Recipe {
	return []recipe.Recipe{preferMinBuiltin, preferMaxBuiltin}
}
