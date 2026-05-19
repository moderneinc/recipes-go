/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"fmt"

	"github.com/moderneinc/recipes-go/recipes-code-quality/diagnostic"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/template"
)

var (
	strS   = template.Expr("s")
	strSub = template.Expr("sub")

	// Positive: strings.Index(s, sub) != -1  ->  strings.Contains(s, sub)
	//           strings.Index(s, sub) >= 0   ->  strings.Contains(s, sub)
	stringsContainsPositive = template.NewRecipe(
		template.RecipeName("org.openrewrite.golang.codequality.PreferStringsContains$Positive"),
		template.WithDisplayName("Prefer strings.Contains (positive)"),
		template.WithBefore(fmt.Sprintf(`strings.Index(%s, %s) != -1`, strS, strSub), template.Imports("strings")),
		template.WithBefore(fmt.Sprintf(`strings.Index(%s, %s) >= 0`, strS, strSub), template.Imports("strings")),
		template.WithAfter(fmt.Sprintf(`strings.Contains(%s, %s)`, strS, strSub), template.Imports("strings")),
		template.WithCaptures(strS, strSub),
	)

	// Negative: strings.Index(s, sub) == -1  ->  !strings.Contains(s, sub)
	//           strings.Index(s, sub) < 0    ->  !strings.Contains(s, sub)
	stringsContainsNegative = template.NewRecipe(
		template.RecipeName("org.openrewrite.golang.codequality.PreferStringsContains$Negative"),
		template.WithDisplayName("Prefer strings.Contains (negative)"),
		template.WithBefore(fmt.Sprintf(`strings.Index(%s, %s) == -1`, strS, strSub), template.Imports("strings")),
		template.WithBefore(fmt.Sprintf(`strings.Index(%s, %s) < 0`, strS, strSub), template.Imports("strings")),
		template.WithAfter(fmt.Sprintf(`!strings.Contains(%s, %s)`, strS, strSub), template.Imports("strings")),
		template.WithCaptures(strS, strSub),
	)
)

// PreferStringsContains replaces comparisons of `strings.Index(s, sub)` against
// -1 or 0 with `strings.Contains(s, sub)` or `!strings.Contains(s, sub)`.
//
// Patterns:
//   - strings.Index(s, sub) != -1  ->  strings.Contains(s, sub)
//   - strings.Index(s, sub) >= 0   ->  strings.Contains(s, sub)
//   - strings.Index(s, sub) == -1  ->  !strings.Contains(s, sub)
//   - strings.Index(s, sub) < 0    ->  !strings.Contains(s, sub)
//
// Staticcheck: S1003
type PreferStringsContains struct {
	recipe.Base
}

func (r *PreferStringsContains) Name() string {
	return "org.openrewrite.golang.codequality.PreferStringsContains"
}
func (r *PreferStringsContains) DisplayName() string {
	return "Prefer strings.Contains over strings.Index comparison"
}
func (r *PreferStringsContains) Description() string {
	return "Replace `strings.Index(s, sub) != -1` and similar patterns with `strings.Contains(s, sub)`."
}
func (r *PreferStringsContains) Tags() []string { return []string{"cleanup", "style"} }

func (r *PreferStringsContains) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{
		{DiagnosticID: "S1003", Tool: diagnostic.Staticcheck, HasFix: true},
	}
}

func (r *PreferStringsContains) RecipeList() []recipe.Recipe {
	return []recipe.Recipe{stringsContainsPositive, stringsContainsNegative}
}
