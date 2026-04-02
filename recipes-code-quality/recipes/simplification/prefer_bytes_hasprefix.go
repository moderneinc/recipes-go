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

var (
	bhpB = template.Expr("bhpB")
	bhpP = template.Expr("bhpP")
)

// PreferBytesHasPrefix replaces `bytes.Index(b, prefix) == 0` with
// `bytes.HasPrefix(b, prefix)`. Staticcheck: S1003
type PreferBytesHasPrefix struct {
	recipe.Base
}

func (r *PreferBytesHasPrefix) Name() string {
	return "org.openrewrite.golang.codequality.PreferBytesHasPrefix"
}
func (r *PreferBytesHasPrefix) DisplayName() string { return "Prefer bytes.HasPrefix" }
func (r *PreferBytesHasPrefix) Description() string {
	return "Replace `bytes.Index(b, prefix) == 0` with `bytes.HasPrefix(b, prefix)` and `bytes.Index(b, prefix) != 0` with `!bytes.HasPrefix(b, prefix)`."
}
func (r *PreferBytesHasPrefix) Tags() []string { return []string{"cleanup", "simplification"} }

func (r *PreferBytesHasPrefix) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{
		{DiagnosticID: "S1003", Tool: diagnostic.Staticcheck, HasFix: true},
	}
}

var preferBytesHasPrefixPositive = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.PreferBytesHasPrefix$Positive"),
	template.WithDisplayName("bytes.Index == 0 → bytes.HasPrefix"),
	template.WithBefore(fmt.Sprintf(`bytes.Index(%s, %s) == 0`, bhpB, bhpP), template.Imports("bytes")),
	template.WithAfter(fmt.Sprintf(`bytes.HasPrefix(%s, %s)`, bhpB, bhpP), template.Imports("bytes")),
	template.WithCaptures(bhpB, bhpP),
)

var preferBytesHasPrefixNegative = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.PreferBytesHasPrefix$Negative"),
	template.WithDisplayName("bytes.Index != 0 → !bytes.HasPrefix"),
	template.WithBefore(fmt.Sprintf(`bytes.Index(%s, %s) != 0`, bhpB, bhpP), template.Imports("bytes")),
	template.WithAfter(fmt.Sprintf(`!bytes.HasPrefix(%s, %s)`, bhpB, bhpP), template.Imports("bytes")),
	template.WithCaptures(bhpB, bhpP),
)

func (r *PreferBytesHasPrefix) RecipeList() []recipe.Recipe {
	return []recipe.Recipe{preferBytesHasPrefixPositive, preferBytesHasPrefixNegative}
}
