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
	crS = template.Expr("crS")
	crR = template.Expr("crR")
)

// PreferStringsContainsRune replaces `strings.IndexRune(s, r) != -1` with
// `strings.ContainsRune(s, r)` and `strings.IndexRune(s, r) == -1` with
// `!strings.ContainsRune(s, r)`.
// Staticcheck: S1003
type PreferStringsContainsRune struct {
	recipe.Base
}

func (r *PreferStringsContainsRune) Name() string {
	return "org.openrewrite.golang.codequality.PreferStringsContainsRune"
}
func (r *PreferStringsContainsRune) DisplayName() string {
	return "Prefer strings.ContainsRune"
}
func (r *PreferStringsContainsRune) Description() string {
	return "Replace `strings.IndexRune(s, r) != -1` with `strings.ContainsRune(s, r)` and `strings.IndexRune(s, r) == -1` with `!strings.ContainsRune(s, r)`."
}
func (r *PreferStringsContainsRune) Tags() []string {
	return []string{"cleanup", "simplification"}
}

func (r *PreferStringsContainsRune) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{
		{DiagnosticID: "S1003", Tool: diagnostic.Staticcheck, HasFix: true},
	}
}

var preferStringsContainsRunePositive = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.PreferStringsContainsRune$Positive"),
	template.WithDisplayName("strings.IndexRune != -1 -> strings.ContainsRune"),
	template.WithBefore(fmt.Sprintf(`strings.IndexRune(%s, %s) != -1`, crS, crR), template.Imports("strings")),
	template.WithAfter(fmt.Sprintf(`strings.ContainsRune(%s, %s)`, crS, crR), template.Imports("strings")),
	template.WithCaptures(crS, crR),
)

var preferStringsContainsRuneGte0 = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.PreferStringsContainsRune$Gte0"),
	template.WithDisplayName("strings.IndexRune >= 0 -> strings.ContainsRune"),
	template.WithBefore(fmt.Sprintf(`strings.IndexRune(%s, %s) >= 0`, crS, crR), template.Imports("strings")),
	template.WithAfter(fmt.Sprintf(`strings.ContainsRune(%s, %s)`, crS, crR), template.Imports("strings")),
	template.WithCaptures(crS, crR),
)

var preferStringsContainsRuneNegative = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.PreferStringsContainsRune$Negative"),
	template.WithDisplayName("strings.IndexRune == -1 -> !strings.ContainsRune"),
	template.WithBefore(fmt.Sprintf(`strings.IndexRune(%s, %s) == -1`, crS, crR), template.Imports("strings")),
	template.WithAfter(fmt.Sprintf(`!strings.ContainsRune(%s, %s)`, crS, crR), template.Imports("strings")),
	template.WithCaptures(crS, crR),
)

var preferStringsContainsRuneLt0 = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.PreferStringsContainsRune$Lt0"),
	template.WithDisplayName("strings.IndexRune < 0 -> !strings.ContainsRune"),
	template.WithBefore(fmt.Sprintf(`strings.IndexRune(%s, %s) < 0`, crS, crR), template.Imports("strings")),
	template.WithAfter(fmt.Sprintf(`!strings.ContainsRune(%s, %s)`, crS, crR), template.Imports("strings")),
	template.WithCaptures(crS, crR),
)

func (r *PreferStringsContainsRune) RecipeList() []recipe.Recipe {
	return []recipe.Recipe{
		preferStringsContainsRunePositive,
		preferStringsContainsRuneGte0,
		preferStringsContainsRuneNegative,
		preferStringsContainsRuneLt0,
	}
}
