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

var atoiS = template.Expr("atoiS")

// PreferStrconvAtoi replaces `strconv.ParseInt(s, 10, 0)` with `strconv.Atoi(s)`.
// Staticcheck: S1030
type PreferStrconvAtoi struct {
	recipe.Base
}

func (r *PreferStrconvAtoi) Name() string {
	return "org.openrewrite.golang.codequality.PreferStrconvAtoi"
}
func (r *PreferStrconvAtoi) DisplayName() string { return "Prefer strconv.Atoi" }
func (r *PreferStrconvAtoi) Description() string {
	return "Replace `strconv.ParseInt(s, 10, 0)` with `strconv.Atoi(s)`."
}
func (r *PreferStrconvAtoi) Tags() []string { return []string{"cleanup", "simplification"} }

func (r *PreferStrconvAtoi) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{
		{DiagnosticID: "S1030", Tool: diagnostic.Staticcheck, HasFix: true},
	}
}

var preferStrconvAtoiImpl = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.PreferStrconvAtoi$Impl"),
	template.WithDisplayName("strconv.ParseInt(s, 10, 0) → strconv.Atoi(s)"),
	template.WithBefore(fmt.Sprintf(`strconv.ParseInt(%s, 10, 0)`, atoiS), template.Imports("strconv")),
	template.WithAfter(fmt.Sprintf(`strconv.Atoi(%s)`, atoiS), template.Imports("strconv")),
	template.WithCaptures(atoiS),
)

func (r *PreferStrconvAtoi) RecipeList() []recipe.Recipe {
	return []recipe.Recipe{preferStrconvAtoiImpl}
}
