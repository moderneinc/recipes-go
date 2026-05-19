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
	scS   = template.Expr("scS")
	scSub = template.Expr("scSub")
)

// PreferStringsContainsOverCount replaces `strings.Count(s, sub) > 0` with
// `strings.Contains(s, sub)` and `strings.Count(s, sub) == 0` with
// `!strings.Contains(s, sub)`.
// Staticcheck: S1003 (partial)
type PreferStringsContainsOverCount struct {
	recipe.Base
}

func (r *PreferStringsContainsOverCount) Name() string {
	return "org.openrewrite.golang.codequality.PreferStringsContainsOverCount"
}
func (r *PreferStringsContainsOverCount) DisplayName() string {
	return "Prefer strings.Contains over strings.Count"
}
func (r *PreferStringsContainsOverCount) Description() string {
	return "Replace `strings.Count(s, sub) > 0` with `strings.Contains(s, sub)` and `strings.Count(s, sub) == 0` with `!strings.Contains(s, sub)`."
}
func (r *PreferStringsContainsOverCount) Tags() []string {
	return []string{"cleanup", "simplification"}
}

func (r *PreferStringsContainsOverCount) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{
		{DiagnosticID: "S1003", Tool: diagnostic.Staticcheck, HasFix: true},
	}
}

var preferStringsContainsOverCountPositive = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.PreferStringsContainsOverCount$Positive"),
	template.WithDisplayName("strings.Count > 0 → strings.Contains"),
	template.WithBefore(fmt.Sprintf(`strings.Count(%s, %s) > 0`, scS, scSub), template.Imports("strings")),
	template.WithAfter(fmt.Sprintf(`strings.Contains(%s, %s)`, scS, scSub), template.Imports("strings")),
	template.WithCaptures(scS, scSub),
)

var preferStringsContainsOverCountNegative = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.PreferStringsContainsOverCount$Negative"),
	template.WithDisplayName("strings.Count == 0 → !strings.Contains"),
	template.WithBefore(fmt.Sprintf(`strings.Count(%s, %s) == 0`, scS, scSub), template.Imports("strings")),
	template.WithAfter(fmt.Sprintf(`!strings.Contains(%s, %s)`, scS, scSub), template.Imports("strings")),
	template.WithCaptures(scS, scSub),
)

func (r *PreferStringsContainsOverCount) RecipeList() []recipe.Recipe {
	return []recipe.Recipe{preferStringsContainsOverCountPositive, preferStringsContainsOverCountNegative}
}
