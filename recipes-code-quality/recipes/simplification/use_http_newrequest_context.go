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
	nrcM = template.Expr("nrcM")
	nrcU = template.Expr("nrcU")
	nrcB = template.Expr("nrcB")
)

// UseHttpNewRequestWithContext replaces `http.NewRequest(method, url, body)` with
// `http.NewRequestWithContext(context.TODO(), method, url, body)` (Go 1.13+).
// Staticcheck: SA1019 (deprecated)
type UseHttpNewRequestWithContext struct {
	recipe.Base
}

func (r *UseHttpNewRequestWithContext) Name() string {
	return "org.openrewrite.golang.codequality.UseHttpNewRequestWithContext"
}
func (r *UseHttpNewRequestWithContext) DisplayName() string {
	return "Use http.NewRequestWithContext"
}
func (r *UseHttpNewRequestWithContext) Description() string {
	return "Replace deprecated `http.NewRequest(method, url, body)` with `http.NewRequestWithContext(context.TODO(), method, url, body)`."
}
func (r *UseHttpNewRequestWithContext) Tags() []string { return []string{"cleanup", "simplification"} }

func (r *UseHttpNewRequestWithContext) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{
		{DiagnosticID: "SA1019", Tool: diagnostic.Staticcheck, HasFix: true},
	}
}

var useHttpNewRequestWithContextImpl = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.UseHttpNewRequestWithContext$Impl"),
	template.WithDisplayName("http.NewRequest → http.NewRequestWithContext"),
	template.WithBefore(fmt.Sprintf(`http.NewRequest(%s, %s, %s)`, nrcM, nrcU, nrcB), template.Imports("net/http")),
	template.WithAfter(fmt.Sprintf(`http.NewRequestWithContext(context.TODO(), %s, %s, %s)`, nrcM, nrcU, nrcB), template.Imports("net/http", "context")),
	template.WithCaptures(nrcM, nrcU, nrcB),
)

func (r *UseHttpNewRequestWithContext) RecipeList() []recipe.Recipe {
	return []recipe.Recipe{useHttpNewRequestWithContextImpl}
}
