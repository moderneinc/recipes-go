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
	snSrc = template.Expr("snSrc")
	snSep = template.Expr("snSep")
)

// SimplifySplitN replaces `strings.SplitN(s, sep, -1)` with `strings.Split(s, sep)`.
// The -1 count means "all substrings", which is the default behavior of strings.Split.
// Staticcheck: S1011
type SimplifySplitN struct {
	recipe.Base
}

func (r *SimplifySplitN) Name() string {
	return "org.openrewrite.golang.codequality.SimplifySplitN"
}
func (r *SimplifySplitN) DisplayName() string {
	return "Simplify strings.SplitN with -1"
}
func (r *SimplifySplitN) Description() string {
	return "Replace `strings.SplitN(s, sep, -1)` with `strings.Split(s, sep)` since -1 means split all."
}
func (r *SimplifySplitN) Tags() []string { return []string{"cleanup", "simplification"} }

func (r *SimplifySplitN) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{
		{DiagnosticID: "S1011", Tool: diagnostic.Staticcheck, HasFix: true},
	}
}

var simplifySplitNImpl = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.SimplifySplitN$Impl"),
	template.WithDisplayName("strings.SplitN -1 \u2192 strings.Split"),
	template.WithBefore(fmt.Sprintf(`strings.SplitN(%s, %s, -1)`, snSrc, snSep), template.Imports("strings")),
	template.WithAfter(fmt.Sprintf(`strings.Split(%s, %s)`, snSrc, snSep), template.Imports("strings")),
	template.WithCaptures(snSrc, snSep),
)

func (r *SimplifySplitN) RecipeList() []recipe.Recipe {
	return []recipe.Recipe{simplifySplitNImpl}
}
