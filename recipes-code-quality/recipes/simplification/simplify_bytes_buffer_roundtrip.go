/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification

import (
	"fmt"

	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/template"
)

var bnbB = template.Expr("bnbB")

// SimplifyBytesBufferRoundtrip replaces `bytes.NewBuffer(b).Bytes()` with `b`.
// Wrapping a byte slice in a buffer only to immediately extract it back is
// a no-op that adds unnecessary overhead.
type SimplifyBytesBufferRoundtrip struct {
	recipe.Base
}

func (r *SimplifyBytesBufferRoundtrip) Name() string {
	return "org.openrewrite.golang.codequality.SimplifyBytesBufferRoundtrip"
}
func (r *SimplifyBytesBufferRoundtrip) DisplayName() string {
	return "Simplify bytes.NewBuffer roundtrip"
}
func (r *SimplifyBytesBufferRoundtrip) Description() string {
	return "Replace `bytes.NewBuffer(b).Bytes()` with `b` since wrapping a byte slice in a buffer only to extract it is a no-op."
}
func (r *SimplifyBytesBufferRoundtrip) Tags() []string {
	return []string{"cleanup", "simplification"}
}

var simplifyBytesBufferRoundtripImpl = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.SimplifyBytesBufferRoundtrip$Impl"),
	template.WithDisplayName("bytes.NewBuffer(b).Bytes() → b"),
	template.WithBefore(fmt.Sprintf(`bytes.NewBuffer(%s).Bytes()`, bnbB), template.Imports("bytes")),
	template.WithAfter(fmt.Sprintf(`%s`, bnbB)),
	template.WithCaptures(bnbB),
)

func (r *SimplifyBytesBufferRoundtrip) RecipeList() []recipe.Recipe {
	return []recipe.Recipe{simplifyBytesBufferRoundtripImpl}
}
