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
	bcrB = template.Expr("bcrB")
	bcrR = template.Expr("bcrR")
)

// PreferBytesContainsRune replaces `bytes.IndexRune(b, r) != -1` with
// `bytes.ContainsRune(b, r)` and `bytes.IndexRune(b, r) == -1` with
// `!bytes.ContainsRune(b, r)`.
// Staticcheck: S1003
type PreferBytesContainsRune struct {
	recipe.Base
}

func (r *PreferBytesContainsRune) Name() string {
	return "org.openrewrite.golang.codequality.PreferBytesContainsRune"
}
func (r *PreferBytesContainsRune) DisplayName() string {
	return "Prefer bytes.ContainsRune"
}
func (r *PreferBytesContainsRune) Description() string {
	return "Replace `bytes.IndexRune(b, r) != -1` with `bytes.ContainsRune(b, r)` and `bytes.IndexRune(b, r) == -1` with `!bytes.ContainsRune(b, r)`."
}
func (r *PreferBytesContainsRune) Tags() []string {
	return []string{"cleanup", "simplification"}
}

func (r *PreferBytesContainsRune) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{
		{DiagnosticID: "S1003", Tool: diagnostic.Staticcheck, HasFix: true},
	}
}

var preferBytesContainsRuneNotEqNeg1 = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.PreferBytesContainsRune$NotEqNeg1"),
	template.WithDisplayName("bytes.IndexRune != -1 -> bytes.ContainsRune"),
	template.WithBefore(fmt.Sprintf(`bytes.IndexRune(%s, %s) != -1`, bcrB, bcrR), template.Imports("bytes")),
	template.WithAfter(fmt.Sprintf(`bytes.ContainsRune(%s, %s)`, bcrB, bcrR), template.Imports("bytes")),
	template.WithCaptures(bcrB, bcrR),
)

var preferBytesContainsRuneGte0 = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.PreferBytesContainsRune$Gte0"),
	template.WithDisplayName("bytes.IndexRune >= 0 -> bytes.ContainsRune"),
	template.WithBefore(fmt.Sprintf(`bytes.IndexRune(%s, %s) >= 0`, bcrB, bcrR), template.Imports("bytes")),
	template.WithAfter(fmt.Sprintf(`bytes.ContainsRune(%s, %s)`, bcrB, bcrR), template.Imports("bytes")),
	template.WithCaptures(bcrB, bcrR),
)

var preferBytesContainsRuneEqNeg1 = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.PreferBytesContainsRune$EqNeg1"),
	template.WithDisplayName("bytes.IndexRune == -1 -> !bytes.ContainsRune"),
	template.WithBefore(fmt.Sprintf(`bytes.IndexRune(%s, %s) == -1`, bcrB, bcrR), template.Imports("bytes")),
	template.WithAfter(fmt.Sprintf(`!bytes.ContainsRune(%s, %s)`, bcrB, bcrR), template.Imports("bytes")),
	template.WithCaptures(bcrB, bcrR),
)

var preferBytesContainsRuneLt0 = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.PreferBytesContainsRune$Lt0"),
	template.WithDisplayName("bytes.IndexRune < 0 -> !bytes.ContainsRune"),
	template.WithBefore(fmt.Sprintf(`bytes.IndexRune(%s, %s) < 0`, bcrB, bcrR), template.Imports("bytes")),
	template.WithAfter(fmt.Sprintf(`!bytes.ContainsRune(%s, %s)`, bcrB, bcrR), template.Imports("bytes")),
	template.WithCaptures(bcrB, bcrR),
)

func (r *PreferBytesContainsRune) RecipeList() []recipe.Recipe {
	return []recipe.Recipe{
		preferBytesContainsRuneNotEqNeg1,
		preferBytesContainsRuneGte0,
		preferBytesContainsRuneEqNeg1,
		preferBytesContainsRuneLt0,
	}
}
