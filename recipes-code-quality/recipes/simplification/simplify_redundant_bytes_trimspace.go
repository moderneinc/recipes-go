/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification

import (
	"fmt"

	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/template"
)

var btsB = template.Expr("btsB")

// SimplifyRedundantBytesTrimSpace replaces `bytes.TrimSpace(bytes.TrimSpace(b))`
// with `bytes.TrimSpace(b)`. The outer call is redundant because TrimSpace is
// idempotent — calling it twice produces the same result as calling it once.
type SimplifyRedundantBytesTrimSpace struct {
	recipe.Base
}

func (r *SimplifyRedundantBytesTrimSpace) Name() string {
	return "org.openrewrite.golang.codequality.SimplifyRedundantBytesTrimSpace"
}
func (r *SimplifyRedundantBytesTrimSpace) DisplayName() string {
	return "Simplify redundant bytes.TrimSpace"
}
func (r *SimplifyRedundantBytesTrimSpace) Description() string {
	return "Replace `bytes.TrimSpace(bytes.TrimSpace(b))` with `bytes.TrimSpace(b)` since TrimSpace is idempotent."
}
func (r *SimplifyRedundantBytesTrimSpace) Tags() []string {
	return []string{"cleanup", "simplification"}
}

var simplifyRedundantBytesTrimSpaceImpl = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.SimplifyRedundantBytesTrimSpace$Impl"),
	template.WithDisplayName("bytes.TrimSpace(bytes.TrimSpace) → bytes.TrimSpace"),
	template.WithBefore(fmt.Sprintf(`bytes.TrimSpace(bytes.TrimSpace(%s))`, btsB), template.Imports("bytes")),
	template.WithAfter(fmt.Sprintf(`bytes.TrimSpace(%s)`, btsB), template.Imports("bytes")),
	template.WithCaptures(btsB),
)

func (r *SimplifyRedundantBytesTrimSpace) RecipeList() []recipe.Recipe {
	return []recipe.Recipe{simplifyRedundantBytesTrimSpaceImpl}
}
