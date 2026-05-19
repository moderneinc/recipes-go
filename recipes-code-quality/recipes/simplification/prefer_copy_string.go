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
	cpDst = template.Expr("cpDst")
	cpSrc = template.Expr("cpSrc")
)

// PreferCopyString replaces `copy(dst, []byte(src))` with `copy(dst, src)`.
// The built-in copy function accepts a string as the source when copying into a
// byte slice, so the explicit []byte conversion is unnecessary.
// Staticcheck: S1030
type PreferCopyString struct {
	recipe.Base
}

func (r *PreferCopyString) Name() string {
	return "org.openrewrite.golang.codequality.PreferCopyString"
}
func (r *PreferCopyString) DisplayName() string { return "Prefer copy from string" }
func (r *PreferCopyString) Description() string {
	return "Replace `copy(dst, []byte(src))` with `copy(dst, src)` since copy accepts a string source."
}
func (r *PreferCopyString) Tags() []string { return []string{"cleanup", "simplification"} }

func (r *PreferCopyString) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{
		{DiagnosticID: "S1030", Tool: diagnostic.Staticcheck, HasFix: true},
	}
}

var preferCopyStringImpl = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.PreferCopyString$Impl"),
	template.WithDisplayName("copy(dst, []byte(src)) → copy(dst, src)"),
	template.WithBefore(fmt.Sprintf(`copy(%s, []byte(%s))`, cpDst, cpSrc)),
	template.WithAfter(fmt.Sprintf(`copy(%s, %s)`, cpDst, cpSrc)),
	template.WithCaptures(cpDst, cpSrc),
)

func (r *PreferCopyString) RecipeList() []recipe.Recipe {
	return []recipe.Recipe{preferCopyStringImpl}
}
