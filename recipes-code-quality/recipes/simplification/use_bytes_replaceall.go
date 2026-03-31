/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification

import (
	"fmt"

	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/template"
)

var (
	braB   = template.Expr("braB")
	braOld = template.Expr("braOld")
	braNew = template.Expr("braNew")
)

// UseBytesReplaceAll replaces `bytes.Replace(b, old, new, -1)` with
// `bytes.ReplaceAll(b, old, new)` (Go 1.12+).
type UseBytesReplaceAll struct {
	recipe.Base
}

func (r *UseBytesReplaceAll) Name() string {
	return "org.openrewrite.golang.codequality.UseBytesReplaceAll"
}
func (r *UseBytesReplaceAll) DisplayName() string {
	return "Use bytes.ReplaceAll"
}
func (r *UseBytesReplaceAll) Description() string {
	return "Replace `bytes.Replace(b, old, new, -1)` with `bytes.ReplaceAll(b, old, new)`."
}
func (r *UseBytesReplaceAll) Tags() []string { return []string{"cleanup", "simplification"} }

var useBytesReplaceAllImpl = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.UseBytesReplaceAll$Impl"),
	template.WithDisplayName("bytes.Replace -1 → bytes.ReplaceAll"),
	template.WithBefore(fmt.Sprintf(`bytes.Replace(%s, %s, %s, -1)`, braB, braOld, braNew), template.Imports("bytes")),
	template.WithAfter(fmt.Sprintf(`bytes.ReplaceAll(%s, %s, %s)`, braB, braOld, braNew), template.Imports("bytes")),
	template.WithCaptures(braB, braOld, braNew),
)

func (r *UseBytesReplaceAll) RecipeList() []recipe.Recipe {
	return []recipe.Recipe{useBytesReplaceAllImpl}
}
