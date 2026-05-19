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
	wsW   = template.Expr("wsW")
	wsStr = template.Expr("wsStr")
)

// PreferIoWriteString replaces `fmt.Fprintf(w, "%s", s)` with `io.WriteString(w, s)`.
// Staticcheck: S1025
type PreferIoWriteString struct {
	recipe.Base
}

func (r *PreferIoWriteString) Name() string {
	return "org.openrewrite.golang.codequality.PreferIoWriteString"
}
func (r *PreferIoWriteString) DisplayName() string { return "Prefer io.WriteString" }
func (r *PreferIoWriteString) Description() string {
	return "Replace `fmt.Fprintf(w, \"%s\", s)` with `io.WriteString(w, s)`."
}
func (r *PreferIoWriteString) Tags() []string { return []string{"cleanup", "simplification"} }

func (r *PreferIoWriteString) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{
		{DiagnosticID: "S1025", Tool: diagnostic.Staticcheck, HasFix: true},
	}
}

var preferIoWriteStringImpl = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.PreferIoWriteString$Impl"),
	template.WithDisplayName(`fmt.Fprintf(w, "%s", s) → io.WriteString(w, s)`),
	template.WithBefore(fmt.Sprintf(`fmt.Fprintf(%s, "%%s", %s)`, wsW, wsStr), template.Imports("fmt")),
	template.WithAfter(fmt.Sprintf(`io.WriteString(%s, %s)`, wsW, wsStr), template.Imports("io")),
	template.WithCaptures(wsW, wsStr),
)

func (r *PreferIoWriteString) RecipeList() []recipe.Recipe {
	return []recipe.Recipe{preferIoWriteStringImpl}
}
