/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"fmt"

	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/template"
)

var heData = template.Expr("heData")

var preferHexEncodingImpl = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.PreferHexEncoding$Impl"),
	template.WithDisplayName("fmt.Sprintf(\"%x\", d) → hex.EncodeToString(d)"),
	template.WithBefore(fmt.Sprintf(`fmt.Sprintf("%%x", %s)`, heData), template.Imports("fmt")),
	template.WithAfter(fmt.Sprintf(`hex.EncodeToString(%s)`, heData), template.Imports("encoding/hex")),
	template.WithCaptures(heData),
)

// PreferHexEncoding replaces `fmt.Sprintf("%x", data)` with
// `hex.EncodeToString(data)` for clearer intent and better performance.
type PreferHexEncoding struct {
	recipe.Base
}

func (r *PreferHexEncoding) Name() string {
	return "org.openrewrite.golang.codequality.PreferHexEncoding"
}
func (r *PreferHexEncoding) DisplayName() string {
	return "Prefer hex.EncodeToString over fmt.Sprintf"
}
func (r *PreferHexEncoding) Description() string {
	return "Replace `fmt.Sprintf(\"%x\", data)` with `hex.EncodeToString(data)` for clearer intent and better performance."
}
func (r *PreferHexEncoding) Tags() []string { return []string{"style", "cleanup"} }

func (r *PreferHexEncoding) RecipeList() []recipe.Recipe {
	return []recipe.Recipe{preferHexEncodingImpl}
}
