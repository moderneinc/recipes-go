/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package redundancy

import (
	"fmt"

	"github.com/moderneinc/recipes-go/code-quality/diagnostic"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/template"
)

var (
	sprintfArg = template.Expr("s")

	removeRedundantSprintfImpl = template.NewRecipe(
		template.RecipeName("org.openrewrite.golang.codequality.RemoveRedundantSprintf"),
		template.WithDisplayName("Remove redundant fmt.Sprintf"),
		template.WithBefore(fmt.Sprintf(`fmt.Sprintf("%%s", %s)`, sprintfArg), template.Imports("fmt")),
		template.WithAfter(fmt.Sprintf(`%s`, sprintfArg)),
		template.WithCaptures(sprintfArg),
	)
)

// RemoveRedundantSprintf replaces `fmt.Sprintf("%s", s)` with just `s`
// when the format string is a single %s and the argument is a string.
// Staticcheck: S1025
type RemoveRedundantSprintf struct {
	recipe.Base
}

func (r *RemoveRedundantSprintf) Name() string {
	return "org.openrewrite.golang.codequality.RemoveRedundantSprintf"
}
func (r *RemoveRedundantSprintf) DisplayName() string { return "Remove redundant fmt.Sprintf" }
func (r *RemoveRedundantSprintf) Description() string {
	return "Replace `fmt.Sprintf(\"%s\", s)` with `s` when the format string is a single %s."
}
func (r *RemoveRedundantSprintf) Tags() []string { return []string{"cleanup", "simplification"} }

func (r *RemoveRedundantSprintf) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{
		{DiagnosticID: "S1025", Tool: diagnostic.Staticcheck, HasFix: true},
	}
}

func (r *RemoveRedundantSprintf) RecipeList() []recipe.Recipe {
	return []recipe.Recipe{removeRedundantSprintfImpl}
}
