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
	bsnSrc = template.Expr("bsnSrc")
	bsnSep = template.Expr("bsnSep")
)

// SimplifyBytesSplitN replaces `bytes.SplitN(b, sep, -1)` with `bytes.Split(b, sep)`.
// The -1 count means "all substrings", which is the default behavior of bytes.Split.
// Staticcheck: S1011
type SimplifyBytesSplitN struct {
	recipe.Base
}

func (r *SimplifyBytesSplitN) Name() string {
	return "org.openrewrite.golang.codequality.SimplifyBytesSplitN"
}
func (r *SimplifyBytesSplitN) DisplayName() string {
	return "Simplify bytes.SplitN with -1"
}
func (r *SimplifyBytesSplitN) Description() string {
	return "Replace `bytes.SplitN(b, sep, -1)` with `bytes.Split(b, sep)` since -1 means split all."
}
func (r *SimplifyBytesSplitN) Tags() []string { return []string{"cleanup", "simplification"} }

func (r *SimplifyBytesSplitN) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{
		{DiagnosticID: "S1011", Tool: diagnostic.Staticcheck, HasFix: true},
	}
}

var simplifyBytesSplitNImpl = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.SimplifyBytesSplitN$Impl"),
	template.WithDisplayName("bytes.SplitN -1 \u2192 bytes.Split"),
	template.WithBefore(fmt.Sprintf(`bytes.SplitN(%s, %s, -1)`, bsnSrc, bsnSep), template.Imports("bytes")),
	template.WithAfter(fmt.Sprintf(`bytes.Split(%s, %s)`, bsnSrc, bsnSep), template.Imports("bytes")),
	template.WithCaptures(bsnSrc, bsnSep),
)

func (r *SimplifyBytesSplitN) RecipeList() []recipe.Recipe {
	return []recipe.Recipe{simplifyBytesSplitNImpl}
}
