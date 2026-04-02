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

var boolX = template.Expr("x")

// simplifyToKeep: patterns where the result is just x
// x == true, true == x, x != false, false != x → x
var simplifyToKeep = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.SimplifyBooleanExpression$Keep"),
	template.WithDisplayName("Simplify boolean (keep)"),
	template.WithBefore(fmt.Sprintf(`%s == true`, boolX)),
	template.WithBefore(fmt.Sprintf(`true == %s`, boolX)),
	template.WithBefore(fmt.Sprintf(`%s != false`, boolX)),
	template.WithBefore(fmt.Sprintf(`false != %s`, boolX)),
	template.WithAfter(fmt.Sprintf(`%s`, boolX)),
	template.WithCaptures(boolX),
)

// simplifyToNegate: patterns where the result is !x
// x == false, false == x, x != true, true != x → !x
var simplifyToNegate = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.SimplifyBooleanExpression$Negate"),
	template.WithDisplayName("Simplify boolean (negate)"),
	template.WithBefore(fmt.Sprintf(`%s == false`, boolX)),
	template.WithBefore(fmt.Sprintf(`false == %s`, boolX)),
	template.WithBefore(fmt.Sprintf(`%s != true`, boolX)),
	template.WithBefore(fmt.Sprintf(`true != %s`, boolX)),
	template.WithAfter(fmt.Sprintf(`!%s`, boolX)),
	template.WithCaptures(boolX),
)

// SimplifyBooleanExpression simplifies boolean comparisons:
//   - `x == true`  -> `x`
//   - `x == false` -> `!x`
//   - `true == x`  -> `x`
//   - `false == x` -> `!x`
//   - `x != true`  -> `!x`
//   - `x != false` -> `x`
//   - `true != x`  -> `!x`
//   - `false != x` -> `x`
//
// Staticcheck: SA4000 (partial)
type SimplifyBooleanExpression struct {
	recipe.Base
}

func (r *SimplifyBooleanExpression) Name() string {
	return "org.openrewrite.golang.codequality.SimplifyBooleanExpression"
}
func (r *SimplifyBooleanExpression) DisplayName() string { return "Simplify boolean expression" }
func (r *SimplifyBooleanExpression) Description() string {
	return "Simplify boolean comparisons like `x == true` to `x` and `x == false` to `!x`."
}
func (r *SimplifyBooleanExpression) Tags() []string { return []string{"cleanup", "simplification"} }

func (r *SimplifyBooleanExpression) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{
		{DiagnosticID: "SA4000", Tool: diagnostic.Staticcheck, HasFix: false},
	}
}

func (r *SimplifyBooleanExpression) RecipeList() []recipe.Recipe {
	return []recipe.Recipe{simplifyToKeep, simplifyToNegate}
}
