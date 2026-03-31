/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification

import (
	"fmt"

	"github.com/moderneinc/recipes-go/code-quality/diagnostic"
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/template"
)

var sprintArg = template.Expr("a")

// simplifySprintToString: fmt.Sprint(#{a}) → fmt.Sprintf("%v", #{a})
// Actually more useful: fmt.Sprintf("%v", #{a}) → fmt.Sprint(#{a})
// Staticcheck: S1025 (partial)

// SimplifyFmtSprintf replaces fmt.Sprintf("%v", x) with fmt.Sprint(x).
// Staticcheck: S1025
type SimplifyFmtSprintf struct {
	recipe.Base
}

func (r *SimplifyFmtSprintf) Name() string {
	return "org.openrewrite.golang.codequality.SimplifyFmtSprintf"
}
func (r *SimplifyFmtSprintf) DisplayName() string { return "Simplify fmt.Sprintf with %%v" }
func (r *SimplifyFmtSprintf) Description() string {
	return "Replace `fmt.Sprintf(\"%v\", x)` with `fmt.Sprint(x)`."
}
func (r *SimplifyFmtSprintf) Tags() []string { return []string{"cleanup", "simplification"} }

func (r *SimplifyFmtSprintf) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{
		{DiagnosticID: "S1025", Tool: diagnostic.Staticcheck, HasFix: true},
	}
}

var simplifyFmtSprintfImpl = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.SimplifyFmtSprintf$Impl"),
	template.WithDisplayName("Simplify fmt.Sprintf with %v"),
	template.WithBefore(fmt.Sprintf(`fmt.Sprintf("%%v", %s)`, sprintArg), template.Imports("fmt")),
	template.WithAfter(fmt.Sprintf(`fmt.Sprint(%s)`, sprintArg), template.Imports("fmt")),
	template.WithCaptures(sprintArg),
)

func (r *SimplifyFmtSprintf) RecipeList() []recipe.Recipe {
	return []recipe.Recipe{simplifyFmtSprintfImpl}
}
