/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification

import (
	"fmt"

	"github.com/moderneinc/recipes-go/code-quality/diagnostic"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/template"
)

var (
	beA = template.Expr("a")
	beB = template.Expr("b")
)

// PreferBytesEqual replaces `bytes.Compare(a, b) == 0` with `bytes.Equal(a, b)`.
// Staticcheck: S1004
type PreferBytesEqual struct {
	recipe.Base
}

func (r *PreferBytesEqual) Name() string {
	return "org.openrewrite.golang.codequality.PreferBytesEqual"
}
func (r *PreferBytesEqual) DisplayName() string { return "Prefer bytes.Equal" }
func (r *PreferBytesEqual) Description() string {
	return "Replace `bytes.Compare(a, b) == 0` with `bytes.Equal(a, b)` and `bytes.Compare(a, b) != 0` with `!bytes.Equal(a, b)`."
}
func (r *PreferBytesEqual) Tags() []string { return []string{"cleanup", "simplification"} }

func (r *PreferBytesEqual) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{
		{DiagnosticID: "S1004", Tool: diagnostic.Staticcheck, HasFix: true},
	}
}

var preferBytesEqualPositive = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.PreferBytesEqual$Positive"),
	template.WithDisplayName("bytes.Compare == 0 → bytes.Equal"),
	template.WithBefore(fmt.Sprintf(`bytes.Compare(%s, %s) == 0`, beA, beB), template.Imports("bytes")),
	template.WithAfter(fmt.Sprintf(`bytes.Equal(%s, %s)`, beA, beB), template.Imports("bytes")),
	template.WithCaptures(beA, beB),
)

var preferBytesEqualNegative = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.PreferBytesEqual$Negative"),
	template.WithDisplayName("bytes.Compare != 0 → !bytes.Equal"),
	template.WithBefore(fmt.Sprintf(`bytes.Compare(%s, %s) != 0`, beA, beB), template.Imports("bytes")),
	template.WithAfter(fmt.Sprintf(`!bytes.Equal(%s, %s)`, beA, beB), template.Imports("bytes")),
	template.WithCaptures(beA, beB),
)

func (r *PreferBytesEqual) RecipeList() []recipe.Recipe {
	return []recipe.Recipe{preferBytesEqualPositive, preferBytesEqualNegative}
}
