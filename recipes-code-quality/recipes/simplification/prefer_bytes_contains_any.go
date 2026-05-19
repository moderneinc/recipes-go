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

var (
	bcaB  = template.Expr("bcaB")
	bcaCh = template.Expr("bcaCh")
)

// PreferBytesContainsAny replaces `bytes.IndexAny(b, chars) != -1` and
// `bytes.IndexAny(b, chars) >= 0` with `bytes.ContainsAny(b, chars)`, and
// `bytes.IndexAny(b, chars) == -1` and `bytes.IndexAny(b, chars) < 0` with
// `!bytes.ContainsAny(b, chars)`.
// Staticcheck: S1003
type PreferBytesContainsAny struct {
	recipe.Base
}

func (r *PreferBytesContainsAny) Name() string {
	return "org.openrewrite.golang.codequality.PreferBytesContainsAny"
}
func (r *PreferBytesContainsAny) DisplayName() string {
	return "Prefer bytes.ContainsAny"
}
func (r *PreferBytesContainsAny) Description() string {
	return "Replace `bytes.IndexAny(b, chars) != -1` with `bytes.ContainsAny(b, chars)` and `bytes.IndexAny(b, chars) == -1` with `!bytes.ContainsAny(b, chars)`."
}
func (r *PreferBytesContainsAny) Tags() []string {
	return []string{"cleanup", "simplification"}
}

func (r *PreferBytesContainsAny) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{
		{DiagnosticID: "S1003", Tool: diagnostic.Staticcheck, HasFix: true},
	}
}

var preferBytesContainsAnyNotEqNeg1 = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.PreferBytesContainsAny$NotEqNeg1"),
	template.WithDisplayName("bytes.IndexAny != -1 -> bytes.ContainsAny"),
	template.WithBefore(fmt.Sprintf(`bytes.IndexAny(%s, %s) != -1`, bcaB, bcaCh), template.Imports("bytes")),
	template.WithAfter(fmt.Sprintf(`bytes.ContainsAny(%s, %s)`, bcaB, bcaCh), template.Imports("bytes")),
	template.WithCaptures(bcaB, bcaCh),
)

var preferBytesContainsAnyGte0 = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.PreferBytesContainsAny$Gte0"),
	template.WithDisplayName("bytes.IndexAny >= 0 -> bytes.ContainsAny"),
	template.WithBefore(fmt.Sprintf(`bytes.IndexAny(%s, %s) >= 0`, bcaB, bcaCh), template.Imports("bytes")),
	template.WithAfter(fmt.Sprintf(`bytes.ContainsAny(%s, %s)`, bcaB, bcaCh), template.Imports("bytes")),
	template.WithCaptures(bcaB, bcaCh),
)

var preferBytesContainsAnyEqNeg1 = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.PreferBytesContainsAny$EqNeg1"),
	template.WithDisplayName("bytes.IndexAny == -1 -> !bytes.ContainsAny"),
	template.WithBefore(fmt.Sprintf(`bytes.IndexAny(%s, %s) == -1`, bcaB, bcaCh), template.Imports("bytes")),
	template.WithAfter(fmt.Sprintf(`!bytes.ContainsAny(%s, %s)`, bcaB, bcaCh), template.Imports("bytes")),
	template.WithCaptures(bcaB, bcaCh),
)

var preferBytesContainsAnyLt0 = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.PreferBytesContainsAny$Lt0"),
	template.WithDisplayName("bytes.IndexAny < 0 -> !bytes.ContainsAny"),
	template.WithBefore(fmt.Sprintf(`bytes.IndexAny(%s, %s) < 0`, bcaB, bcaCh), template.Imports("bytes")),
	template.WithAfter(fmt.Sprintf(`!bytes.ContainsAny(%s, %s)`, bcaB, bcaCh), template.Imports("bytes")),
	template.WithCaptures(bcaB, bcaCh),
)

func (r *PreferBytesContainsAny) RecipeList() []recipe.Recipe {
	return []recipe.Recipe{
		preferBytesContainsAnyNotEqNeg1,
		preferBytesContainsAnyGte0,
		preferBytesContainsAnyEqNeg1,
		preferBytesContainsAnyLt0,
	}
}
