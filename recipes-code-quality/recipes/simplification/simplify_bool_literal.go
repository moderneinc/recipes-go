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

var dblX = template.Expr("dblX")

// SimplifyDoubleNegation replaces `!!x` with `x` (double boolean negation).
// Staticcheck: S1001
type SimplifyDoubleNegation struct {
	recipe.Base
}

func (r *SimplifyDoubleNegation) Name() string {
	return "org.openrewrite.golang.codequality.SimplifyDoubleNegation"
}
func (r *SimplifyDoubleNegation) DisplayName() string { return "Simplify double negation" }
func (r *SimplifyDoubleNegation) Description() string {
	return "Replace `!!x` with `x` to remove redundant double boolean negation."
}
func (r *SimplifyDoubleNegation) Tags() []string { return []string{"cleanup", "simplification"} }

func (r *SimplifyDoubleNegation) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{
		{DiagnosticID: "S1001", Tool: diagnostic.Staticcheck, HasFix: false},
	}
}

var simplifyDoubleNegationImpl = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.SimplifyDoubleNegation$Impl"),
	template.WithDisplayName("!!x → x"),
	template.WithBefore(fmt.Sprintf(`!!%s`, dblX)),
	template.WithAfter(fmt.Sprintf(`%s`, dblX)),
	template.WithCaptures(dblX),
)

func (r *SimplifyDoubleNegation) RecipeList() []recipe.Recipe {
	return []recipe.Recipe{simplifyDoubleNegationImpl}
}
