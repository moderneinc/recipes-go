/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"fmt"

	"github.com/moderneinc/recipes-go/code-quality/diagnostic"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/template"
)

var (
	mapKey = template.Expr("k")
	mapVal = template.Expr("v")
)

// PreferMakeForEmptyMap replaces `map[K]V{}` with `make(map[K]V)` for
// empty map initialization. This is the idiomatic Go style when the map
// will be populated later.
// golangci-lint: gocritic (emptyMapLiteral)
type PreferMakeForEmptyMap struct {
	recipe.Base
}

func (r *PreferMakeForEmptyMap) Name() string {
	return "org.openrewrite.golang.codequality.PreferMakeForEmptyMap"
}
func (r *PreferMakeForEmptyMap) DisplayName() string { return "Prefer make() for empty maps" }
func (r *PreferMakeForEmptyMap) Description() string {
	return "Replace empty map literal `map[K]V{}` with `make(map[K]V)` for clarity."
}
func (r *PreferMakeForEmptyMap) Tags() []string { return []string{"style"} }

func (r *PreferMakeForEmptyMap) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{
		{DiagnosticID: "emptyMapLiteral", Tool: diagnostic.GolangciLint, HasFix: false},
	}
}

// Note: This pattern requires matching composite literals with type expressions,
// which the current template system doesn't support well for generic map types.
// Keeping as a search-only placeholder for now.
func (r *PreferMakeForEmptyMap) RecipeList() []recipe.Recipe { return nil }

// --- Simpler template-expressible recipes ---

var (
	newS1 = template.Expr("s")
	newN1 = template.Expr("n")
)

// PreferStringsEqualFold replaces `strings.ToLower(s) == strings.ToLower(n)`
// with `strings.EqualFold(s, n)` for case-insensitive comparison.
// Staticcheck: SA6005
type PreferStringsEqualFold struct {
	recipe.Base
}

func (r *PreferStringsEqualFold) Name() string {
	return "org.openrewrite.golang.codequality.PreferStringsEqualFold"
}
func (r *PreferStringsEqualFold) DisplayName() string { return "Prefer strings.EqualFold" }
func (r *PreferStringsEqualFold) Description() string {
	return "Replace `strings.ToLower(s) == strings.ToLower(n)` with `strings.EqualFold(s, n)`."
}
func (r *PreferStringsEqualFold) Tags() []string { return []string{"cleanup", "simplification"} }

func (r *PreferStringsEqualFold) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{
		{DiagnosticID: "SA6005", Tool: diagnostic.Staticcheck, HasFix: true},
	}
}

var preferEqualFoldLower = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.PreferStringsEqualFold$Lower"),
	template.WithDisplayName("ToLower == ToLower → EqualFold"),
	template.WithBefore(
		fmt.Sprintf(`strings.ToLower(%s) == strings.ToLower(%s)`, newS1, newN1),
		template.Imports("strings"),
	),
	template.WithAfter(
		fmt.Sprintf(`strings.EqualFold(%s, %s)`, newS1, newN1),
		template.Imports("strings"),
	),
	template.WithCaptures(newS1, newN1),
)

var preferEqualFoldUpper = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.PreferStringsEqualFold$Upper"),
	template.WithDisplayName("ToUpper == ToUpper → EqualFold"),
	template.WithBefore(
		fmt.Sprintf(`strings.ToUpper(%s) == strings.ToUpper(%s)`, newS1, newN1),
		template.Imports("strings"),
	),
	template.WithAfter(
		fmt.Sprintf(`strings.EqualFold(%s, %s)`, newS1, newN1),
		template.Imports("strings"),
	),
	template.WithCaptures(newS1, newN1),
)

func (r *PreferStringsEqualFold) RecipeList() []recipe.Recipe {
	return []recipe.Recipe{preferEqualFoldLower, preferEqualFoldUpper}
}
