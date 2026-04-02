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

var snrS = template.Expr("snrS")

// PreferStringsNewReader replaces `bytes.NewReader([]byte(s))` with
// `strings.NewReader(s)`. When the source is already a string, converting
// it to []byte only to wrap it in a bytes.Reader is wasteful; strings.NewReader
// avoids the allocation.
// Staticcheck: S1036
type PreferStringsNewReader struct {
	recipe.Base
}

func (r *PreferStringsNewReader) Name() string {
	return "org.openrewrite.golang.codequality.PreferStringsNewReader"
}
func (r *PreferStringsNewReader) DisplayName() string { return "Prefer strings.NewReader" }
func (r *PreferStringsNewReader) Description() string {
	return "Replace `bytes.NewReader([]byte(s))` with `strings.NewReader(s)` to avoid an unnecessary string-to-byte-slice conversion."
}
func (r *PreferStringsNewReader) Tags() []string { return []string{"cleanup", "simplification"} }

func (r *PreferStringsNewReader) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{
		{DiagnosticID: "S1036", Tool: diagnostic.Staticcheck, HasFix: true},
	}
}

var preferStringsNewReaderImpl = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.PreferStringsNewReader$Impl"),
	template.WithDisplayName("bytes.NewReader([]byte) → strings.NewReader"),
	template.WithBefore(fmt.Sprintf(`bytes.NewReader([]byte(%s))`, snrS), template.Imports("bytes")),
	template.WithAfter(fmt.Sprintf(`strings.NewReader(%s)`, snrS), template.Imports("strings")),
	template.WithCaptures(snrS),
)

func (r *PreferStringsNewReader) RecipeList() []recipe.Recipe {
	return []recipe.Recipe{preferStringsNewReaderImpl}
}
