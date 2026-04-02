/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package performance

import (
	"fmt"

	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/template"
)

var (
	bbufB = template.Expr("bbufB")

	preferBytesBufferStringImpl = template.NewRecipe(
		template.RecipeName("org.openrewrite.golang.codequality.PreferBytesBufferString$Impl"),
		template.WithDisplayName("Prefer buf.String() over string(buf.Bytes())"),
		template.WithBefore(fmt.Sprintf(`string(%s.Bytes())`, bbufB)),
		template.WithAfter(fmt.Sprintf(`%s.String()`, bbufB)),
		template.WithCaptures(bbufB),
	)
)

// PreferBytesBufferString replaces `string(buf.Bytes())` with `buf.String()`
// for better performance and readability when working with bytes.Buffer.
type PreferBytesBufferString struct {
	recipe.Base
}

func (r *PreferBytesBufferString) Name() string {
	return "org.openrewrite.golang.codequality.PreferBytesBufferString"
}
func (r *PreferBytesBufferString) DisplayName() string {
	return "Prefer buf.String() over string(buf.Bytes())"
}
func (r *PreferBytesBufferString) Description() string {
	return "Replace `string(buf.Bytes())` with `buf.String()` for better performance and readability."
}
func (r *PreferBytesBufferString) Tags() []string { return []string{"performance"} }

func (r *PreferBytesBufferString) RecipeList() []recipe.Recipe {
	return []recipe.Recipe{preferBytesBufferStringImpl}
}
