/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification

import (
	"fmt"

	"github.com/moderneinc/recipes-go/recipes-code-quality/diagnostic"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/template"
)

var benB = template.Expr("benB")

// SimplifyBytesEqualNil replaces `bytes.Equal(b, nil)` and `bytes.Equal(nil, b)`
// with `len(b) == 0`.
// Staticcheck: S1003
type SimplifyBytesEqualNil struct {
	recipe.Base
}

func (r *SimplifyBytesEqualNil) Name() string {
	return "org.openrewrite.golang.codequality.SimplifyBytesEqualNil"
}
func (r *SimplifyBytesEqualNil) DisplayName() string {
	return "Simplify bytes.Equal nil check"
}
func (r *SimplifyBytesEqualNil) Description() string {
	return "Replace `bytes.Equal(b, nil)` and `bytes.Equal(nil, b)` with `len(b) == 0`."
}
func (r *SimplifyBytesEqualNil) Tags() []string {
	return []string{"cleanup", "simplification"}
}

func (r *SimplifyBytesEqualNil) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{
		{DiagnosticID: "S1003", Tool: diagnostic.Staticcheck, HasFix: true},
	}
}

var simplifyBytesEqualNilRight = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.SimplifyBytesEqualNil$Right"),
	template.WithDisplayName("bytes.Equal(b, nil) -> len(b) == 0"),
	template.WithBefore(fmt.Sprintf(`bytes.Equal(%s, nil)`, benB), template.Imports("bytes")),
	template.WithAfter(fmt.Sprintf(`len(%s) == 0`, benB)),
	template.WithCaptures(benB),
)

var simplifyBytesEqualNilLeft = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.SimplifyBytesEqualNil$Left"),
	template.WithDisplayName("bytes.Equal(nil, b) -> len(b) == 0"),
	template.WithBefore(fmt.Sprintf(`bytes.Equal(nil, %s)`, benB), template.Imports("bytes")),
	template.WithAfter(fmt.Sprintf(`len(%s) == 0`, benB)),
	template.WithCaptures(benB),
)

func (r *SimplifyBytesEqualNil) RecipeList() []recipe.Recipe {
	return []recipe.Recipe{simplifyBytesEqualNilRight, simplifyBytesEqualNilLeft}
}
