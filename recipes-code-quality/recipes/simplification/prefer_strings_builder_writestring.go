/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification

import (
	"fmt"

	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/template"
)

var (
	sbwB = template.Expr("sbwB")
	sbwS = template.Expr("sbwS")
)

// PreferStringsBuilderWriteString replaces `fmt.Fprintf(&b, "%s", s)` with
// `b.WriteString(s)` when writing to a strings.Builder. Staticcheck: S1038
type PreferStringsBuilderWriteString struct {
	recipe.Base
}

func (r *PreferStringsBuilderWriteString) Name() string {
	return "org.openrewrite.golang.codequality.PreferStringsBuilderWriteString"
}
func (r *PreferStringsBuilderWriteString) DisplayName() string {
	return "Prefer strings.Builder WriteString"
}
func (r *PreferStringsBuilderWriteString) Description() string {
	return "Replace `fmt.Fprintf(&b, \"%s\", s)` with `b.WriteString(s)` for more efficient string building."
}
func (r *PreferStringsBuilderWriteString) Tags() []string {
	return []string{"cleanup", "simplification"}
}

var preferStringsBuilderWriteStringImpl = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.PreferStringsBuilderWriteString$Impl"),
	template.WithDisplayName(`fmt.Fprintf(&b, "%s", s) -> b.WriteString(s)`),
	template.WithBefore(fmt.Sprintf(`fmt.Fprintf(&%s, "%%s", %s)`, sbwB, sbwS), template.Imports("fmt")),
	template.WithAfter(fmt.Sprintf(`%s.WriteString(%s)`, sbwB, sbwS)),
	template.WithCaptures(sbwB, sbwS),
)

func (r *PreferStringsBuilderWriteString) RecipeList() []recipe.Recipe {
	return []recipe.Recipe{preferStringsBuilderWriteStringImpl}
}
