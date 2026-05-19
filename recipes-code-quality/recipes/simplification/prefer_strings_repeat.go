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
	scA = template.Expr("scA")
	scB = template.Expr("scB")
)

// SimplifySprintfConcat replaces `fmt.Sprintf("%s%s", a, b)` with `a + b`.
// Using fmt.Sprintf for simple string concatenation is unnecessary overhead.
type SimplifySprintfConcat struct {
	recipe.Base
}

func (r *SimplifySprintfConcat) Name() string {
	return "org.openrewrite.golang.codequality.SimplifySprintfConcat"
}
func (r *SimplifySprintfConcat) DisplayName() string { return "Simplify fmt.Sprintf string concat" }
func (r *SimplifySprintfConcat) Description() string {
	return "Replace `fmt.Sprintf(\"%s%s\", a, b)` with `a + b` for simple string concatenation."
}
func (r *SimplifySprintfConcat) Tags() []string { return []string{"cleanup", "simplification"} }

func (r *SimplifySprintfConcat) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{}
}

var simplifySprintfConcatImpl = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.SimplifySprintfConcat$Impl"),
	template.WithDisplayName("fmt.Sprintf(\"%s%s\", a, b) → a + b"),
	template.WithBefore(fmt.Sprintf(`fmt.Sprintf("%%s%%s", %s, %s)`, scA, scB), template.Imports("fmt")),
	template.WithAfter(fmt.Sprintf(`%s + %s`, scA, scB)),
	template.WithCaptures(scA, scB),
)

func (r *SimplifySprintfConcat) RecipeList() []recipe.Recipe {
	return []recipe.Recipe{simplifySprintfConcatImpl}
}
