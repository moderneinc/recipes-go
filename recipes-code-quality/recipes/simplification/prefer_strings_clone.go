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

var tlnS = template.Expr("tlnS")

// SimplifyTrimLeftNoop removes no-op calls to strings.TrimLeft and
// strings.TrimRight where the cutset is an empty string.
// `strings.TrimLeft(s, "")` and `strings.TrimRight(s, "")` both return s
// unchanged, so the call can be replaced by s directly.
// Staticcheck: SA1024
type SimplifyTrimLeftNoop struct {
	recipe.Base
}

func (r *SimplifyTrimLeftNoop) Name() string {
	return "org.openrewrite.golang.codequality.SimplifyTrimLeftNoop"
}
func (r *SimplifyTrimLeftNoop) DisplayName() string { return "Simplify no-op TrimLeft/TrimRight" }
func (r *SimplifyTrimLeftNoop) Description() string {
	return "Replace `strings.TrimLeft(s, \"\")` and `strings.TrimRight(s, \"\")` with `s` since trimming with an empty cutset is a no-op."
}
func (r *SimplifyTrimLeftNoop) Tags() []string { return []string{"cleanup", "simplification"} }

func (r *SimplifyTrimLeftNoop) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{
		{DiagnosticID: "SA1024", Tool: diagnostic.Staticcheck, HasFix: false},
	}
}

var simplifyTrimLeftNoopImpl = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.SimplifyTrimLeftNoop$TrimLeft"),
	template.WithDisplayName("strings.TrimLeft(s, \"\") → s"),
	template.WithBefore(fmt.Sprintf(`strings.TrimLeft(%s, "")`, tlnS), template.Imports("strings")),
	template.WithAfter(fmt.Sprintf(`%s`, tlnS)),
	template.WithCaptures(tlnS),
)

var simplifyTrimRightNoopImpl = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.SimplifyTrimLeftNoop$TrimRight"),
	template.WithDisplayName("strings.TrimRight(s, \"\") → s"),
	template.WithBefore(fmt.Sprintf(`strings.TrimRight(%s, "")`, tlnS), template.Imports("strings")),
	template.WithAfter(fmt.Sprintf(`%s`, tlnS)),
	template.WithCaptures(tlnS),
)

func (r *SimplifyTrimLeftNoop) RecipeList() []recipe.Recipe {
	return []recipe.Recipe{simplifyTrimLeftNoopImpl, simplifyTrimRightNoopImpl}
}
