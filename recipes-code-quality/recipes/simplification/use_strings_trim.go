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

var tsS = template.Expr("tsS")

// SimplifyRedundantTrimSpace replaces `strings.TrimSpace(strings.TrimSpace(s))`
// with `strings.TrimSpace(s)` (double TrimSpace is redundant).
type SimplifyRedundantTrimSpace struct {
	recipe.Base
}

func (r *SimplifyRedundantTrimSpace) Name() string {
	return "org.openrewrite.golang.codequality.SimplifyRedundantTrimSpace"
}
func (r *SimplifyRedundantTrimSpace) DisplayName() string {
	return "Simplify redundant TrimSpace"
}
func (r *SimplifyRedundantTrimSpace) Description() string {
	return "Replace `strings.TrimSpace(strings.TrimSpace(s))` with `strings.TrimSpace(s)`."
}
func (r *SimplifyRedundantTrimSpace) Tags() []string { return []string{"cleanup", "simplification"} }

func (r *SimplifyRedundantTrimSpace) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{}
}

var simplifyRedundantTrimSpaceImpl = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.SimplifyRedundantTrimSpace$Impl"),
	template.WithDisplayName("strings.TrimSpace(strings.TrimSpace) → strings.TrimSpace"),
	template.WithBefore(fmt.Sprintf(`strings.TrimSpace(strings.TrimSpace(%s))`, tsS), template.Imports("strings")),
	template.WithAfter(fmt.Sprintf(`strings.TrimSpace(%s)`, tsS), template.Imports("strings")),
	template.WithCaptures(tsS),
)

func (r *SimplifyRedundantTrimSpace) RecipeList() []recipe.Recipe {
	return []recipe.Recipe{simplifyRedundantTrimSpaceImpl}
}
