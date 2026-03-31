/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification

import (
	"github.com/moderneinc/recipes-go/code-quality/diagnostic"
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/template"
)

// PreferIoDiscard replaces `ioutil.Discard` with `io.Discard` (Go 1.16+).
// Staticcheck: SA1019 (deprecated)
type PreferIoDiscard struct {
	recipe.Base
}

func (r *PreferIoDiscard) Name() string {
	return "org.openrewrite.golang.codequality.PreferIoDiscard"
}
func (r *PreferIoDiscard) DisplayName() string {
	return "Prefer io.Discard"
}
func (r *PreferIoDiscard) Description() string {
	return "Replace deprecated `ioutil.Discard` with `io.Discard` (Go 1.16+)."
}
func (r *PreferIoDiscard) Tags() []string { return []string{"cleanup", "simplification"} }

func (r *PreferIoDiscard) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{
		{DiagnosticID: "SA1019", Tool: diagnostic.Staticcheck, HasFix: true},
	}
}

var preferIoDiscardImpl = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.PreferIoDiscard$Impl"),
	template.WithDisplayName("ioutil.Discard → io.Discard"),
	template.WithBefore(`ioutil.Discard`, template.Imports("io/ioutil")),
	template.WithAfter(`io.Discard`, template.Imports("io")),
)

func (r *PreferIoDiscard) RecipeList() []recipe.Recipe {
	return []recipe.Recipe{preferIoDiscardImpl}
}
