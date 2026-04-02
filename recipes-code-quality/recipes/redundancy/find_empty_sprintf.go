/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package redundancy

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/template"
)

// FindEmptyFmtSprintf replaces `fmt.Sprintf("")` calls where the format string is
// empty and there are no additional arguments with `""`, since the call always
// returns "" and is redundant.
type FindEmptyFmtSprintf struct {
	recipe.Base
}

func (r *FindEmptyFmtSprintf) Name() string {
	return "org.openrewrite.golang.codequality.FindEmptyFmtSprintf"
}
func (r *FindEmptyFmtSprintf) DisplayName() string { return "Remove empty fmt.Sprintf" }
func (r *FindEmptyFmtSprintf) Description() string {
	return "Replace `fmt.Sprintf(\"\")` calls with an empty format string and no args with `\"\"`."
}
func (r *FindEmptyFmtSprintf) Tags() []string { return []string{"cleanup", "redundancy"} }

var removeEmptyFmtSprintfImpl = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.FindEmptyFmtSprintf$Impl"),
	template.WithDisplayName("fmt.Sprintf(\"\") -> \"\""),
	template.WithBefore(`fmt.Sprintf("")`, template.Imports("fmt")),
	template.WithAfter(`""`),
)

func (r *FindEmptyFmtSprintf) RecipeList() []recipe.Recipe {
	return []recipe.Recipe{removeEmptyFmtSprintfImpl}
}
