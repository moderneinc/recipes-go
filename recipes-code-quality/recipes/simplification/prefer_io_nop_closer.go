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

var ncR = template.Expr("ncR")

// PreferIoNopCloser replaces `ioutil.NopCloser(r)` with `io.NopCloser(r)` (Go 1.16+).
// Staticcheck: SA1019 (deprecated)
type PreferIoNopCloser struct {
	recipe.Base
}

func (r *PreferIoNopCloser) Name() string {
	return "org.openrewrite.golang.codequality.PreferIoNopCloser"
}
func (r *PreferIoNopCloser) DisplayName() string {
	return "Prefer io.NopCloser"
}
func (r *PreferIoNopCloser) Description() string {
	return "Replace deprecated `ioutil.NopCloser(r)` with `io.NopCloser(r)` (Go 1.16+)."
}
func (r *PreferIoNopCloser) Tags() []string { return []string{"cleanup", "simplification"} }

func (r *PreferIoNopCloser) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{
		{DiagnosticID: "SA1019", Tool: diagnostic.Staticcheck, HasFix: true},
	}
}

var preferIoNopCloserImpl = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.PreferIoNopCloser$Impl"),
	template.WithDisplayName("ioutil.NopCloser → io.NopCloser"),
	template.WithBefore(fmt.Sprintf(`ioutil.NopCloser(%s)`, ncR), template.Imports("io/ioutil")),
	template.WithAfter(fmt.Sprintf(`io.NopCloser(%s)`, ncR), template.Imports("io")),
	template.WithCaptures(ncR),
)

func (r *PreferIoNopCloser) RecipeList() []recipe.Recipe {
	return []recipe.Recipe{preferIoNopCloserImpl}
}
