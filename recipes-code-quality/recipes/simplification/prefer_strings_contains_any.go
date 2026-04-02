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
	caS  = template.Expr("caS")
	caCh = template.Expr("caCh")
)

// PreferStringsContainsAny replaces `strings.IndexAny(s, chars) != -1` and
// `strings.IndexAny(s, chars) >= 0` with `strings.ContainsAny(s, chars)`, and
// `strings.IndexAny(s, chars) == -1` and `strings.IndexAny(s, chars) < 0` with
// `!strings.ContainsAny(s, chars)`.
// Staticcheck: S1003
type PreferStringsContainsAny struct {
	recipe.Base
}

func (r *PreferStringsContainsAny) Name() string {
	return "org.openrewrite.golang.codequality.PreferStringsContainsAny"
}
func (r *PreferStringsContainsAny) DisplayName() string {
	return "Prefer strings.ContainsAny"
}
func (r *PreferStringsContainsAny) Description() string {
	return "Replace `strings.IndexAny(s, chars) != -1` with `strings.ContainsAny(s, chars)` and `strings.IndexAny(s, chars) == -1` with `!strings.ContainsAny(s, chars)`."
}
func (r *PreferStringsContainsAny) Tags() []string {
	return []string{"cleanup", "simplification"}
}

func (r *PreferStringsContainsAny) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{
		{DiagnosticID: "S1003", Tool: diagnostic.Staticcheck, HasFix: true},
	}
}

var preferStringsContainsAnyNotEqNeg1 = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.PreferStringsContainsAny$NotEqNeg1"),
	template.WithDisplayName("strings.IndexAny != -1 -> strings.ContainsAny"),
	template.WithBefore(fmt.Sprintf(`strings.IndexAny(%s, %s) != -1`, caS, caCh), template.Imports("strings")),
	template.WithAfter(fmt.Sprintf(`strings.ContainsAny(%s, %s)`, caS, caCh), template.Imports("strings")),
	template.WithCaptures(caS, caCh),
)

var preferStringsContainsAnyGte0 = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.PreferStringsContainsAny$Gte0"),
	template.WithDisplayName("strings.IndexAny >= 0 -> strings.ContainsAny"),
	template.WithBefore(fmt.Sprintf(`strings.IndexAny(%s, %s) >= 0`, caS, caCh), template.Imports("strings")),
	template.WithAfter(fmt.Sprintf(`strings.ContainsAny(%s, %s)`, caS, caCh), template.Imports("strings")),
	template.WithCaptures(caS, caCh),
)

var preferStringsContainsAnyEqNeg1 = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.PreferStringsContainsAny$EqNeg1"),
	template.WithDisplayName("strings.IndexAny == -1 -> !strings.ContainsAny"),
	template.WithBefore(fmt.Sprintf(`strings.IndexAny(%s, %s) == -1`, caS, caCh), template.Imports("strings")),
	template.WithAfter(fmt.Sprintf(`!strings.ContainsAny(%s, %s)`, caS, caCh), template.Imports("strings")),
	template.WithCaptures(caS, caCh),
)

var preferStringsContainsAnyLt0 = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.PreferStringsContainsAny$Lt0"),
	template.WithDisplayName("strings.IndexAny < 0 -> !strings.ContainsAny"),
	template.WithBefore(fmt.Sprintf(`strings.IndexAny(%s, %s) < 0`, caS, caCh), template.Imports("strings")),
	template.WithAfter(fmt.Sprintf(`!strings.ContainsAny(%s, %s)`, caS, caCh), template.Imports("strings")),
	template.WithCaptures(caS, caCh),
)

func (r *PreferStringsContainsAny) RecipeList() []recipe.Recipe {
	return []recipe.Recipe{
		preferStringsContainsAnyNotEqNeg1,
		preferStringsContainsAnyGte0,
		preferStringsContainsAnyEqNeg1,
		preferStringsContainsAnyLt0,
	}
}
