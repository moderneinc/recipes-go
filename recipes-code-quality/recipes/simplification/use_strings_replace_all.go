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
	raSrc = template.Expr("raSrc")
	raOld = template.Expr("raOld")
	raNew = template.Expr("raNew")
)

// UseStringsReplaceAll replaces `strings.Replace(s, old, new, -1)` with
// `strings.ReplaceAll(s, old, new)` (Go 1.12+).
// Staticcheck: S1017
type UseStringsReplaceAll struct {
	recipe.Base
}

func (r *UseStringsReplaceAll) Name() string {
	return "org.openrewrite.golang.codequality.UseStringsReplaceAll"
}
func (r *UseStringsReplaceAll) DisplayName() string {
	return "Use strings.ReplaceAll"
}
func (r *UseStringsReplaceAll) Description() string {
	return "Replace `strings.Replace(s, old, new, -1)` with `strings.ReplaceAll(s, old, new)`."
}
func (r *UseStringsReplaceAll) Tags() []string { return []string{"cleanup", "simplification"} }

func (r *UseStringsReplaceAll) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{
		{DiagnosticID: "S1017", Tool: diagnostic.Staticcheck, HasFix: true},
	}
}

var useStringsReplaceAllImpl = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.UseStringsReplaceAll$Impl"),
	template.WithDisplayName("strings.Replace -1 → strings.ReplaceAll"),
	template.WithBefore(fmt.Sprintf(`strings.Replace(%s, %s, %s, -1)`, raSrc, raOld, raNew), template.Imports("strings")),
	template.WithAfter(fmt.Sprintf(`strings.ReplaceAll(%s, %s, %s)`, raSrc, raOld, raNew), template.Imports("strings")),
	template.WithCaptures(raSrc, raOld, raNew),
)

func (r *UseStringsReplaceAll) RecipeList() []recipe.Recipe {
	return []recipe.Recipe{useStringsReplaceAllImpl}
}
