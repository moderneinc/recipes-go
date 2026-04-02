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

var raR = template.Expr("raR")

// PreferIoReadAll replaces `ioutil.ReadAll(r)` with `io.ReadAll(r)` (Go 1.16+).
// Staticcheck: SA1019 (deprecated)
type PreferIoReadAll struct {
	recipe.Base
}

func (r *PreferIoReadAll) Name() string {
	return "org.openrewrite.golang.codequality.PreferIoReadAll"
}
func (r *PreferIoReadAll) DisplayName() string {
	return "Prefer io.ReadAll"
}
func (r *PreferIoReadAll) Description() string {
	return "Replace deprecated `ioutil.ReadAll(r)` with `io.ReadAll(r)` (Go 1.16+)."
}
func (r *PreferIoReadAll) Tags() []string { return []string{"cleanup", "simplification"} }

func (r *PreferIoReadAll) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{
		{DiagnosticID: "SA1019", Tool: diagnostic.Staticcheck, HasFix: true},
	}
}

var preferIoReadAllImpl = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.PreferIoReadAll$Impl"),
	template.WithDisplayName("ioutil.ReadAll → io.ReadAll"),
	template.WithBefore(fmt.Sprintf(`ioutil.ReadAll(%s)`, raR), template.Imports("io/ioutil")),
	template.WithAfter(fmt.Sprintf(`io.ReadAll(%s)`, raR), template.Imports("io")),
	template.WithCaptures(raR),
)

func (r *PreferIoReadAll) RecipeList() []recipe.Recipe {
	return []recipe.Recipe{preferIoReadAllImpl}
}
