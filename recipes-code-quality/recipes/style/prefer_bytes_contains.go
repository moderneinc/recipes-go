/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"fmt"

	"github.com/moderneinc/recipes-go/code-quality/diagnostic"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/template"
)

var (
	bytB   = template.Expr("b")
	bytSub = template.Expr("sub2")

	// Positive: bytes.Index(b, sub) != -1  ->  bytes.Contains(b, sub)
	//           bytes.Index(b, sub) >= 0   ->  bytes.Contains(b, sub)
	bytesContainsPositive = template.NewRecipe(
		template.RecipeName("org.openrewrite.golang.codequality.PreferBytesContains$Positive"),
		template.WithDisplayName("Prefer bytes.Contains (positive)"),
		template.WithBefore(fmt.Sprintf(`bytes.Index(%s, %s) != -1`, bytB, bytSub), template.Imports("bytes")),
		template.WithBefore(fmt.Sprintf(`bytes.Index(%s, %s) >= 0`, bytB, bytSub), template.Imports("bytes")),
		template.WithAfter(fmt.Sprintf(`bytes.Contains(%s, %s)`, bytB, bytSub), template.Imports("bytes")),
		template.WithCaptures(bytB, bytSub),
	)

	// Negative: bytes.Index(b, sub) == -1  ->  !bytes.Contains(b, sub)
	//           bytes.Index(b, sub) < 0    ->  !bytes.Contains(b, sub)
	bytesContainsNegative = template.NewRecipe(
		template.RecipeName("org.openrewrite.golang.codequality.PreferBytesContains$Negative"),
		template.WithDisplayName("Prefer bytes.Contains (negative)"),
		template.WithBefore(fmt.Sprintf(`bytes.Index(%s, %s) == -1`, bytB, bytSub), template.Imports("bytes")),
		template.WithBefore(fmt.Sprintf(`bytes.Index(%s, %s) < 0`, bytB, bytSub), template.Imports("bytes")),
		template.WithAfter(fmt.Sprintf(`!bytes.Contains(%s, %s)`, bytB, bytSub), template.Imports("bytes")),
		template.WithCaptures(bytB, bytSub),
	)
)

// PreferBytesContains replaces comparisons of `bytes.Index(b, sub)` against
// -1 or 0 with `bytes.Contains(b, sub)` or `!bytes.Contains(b, sub)`.
//
// Patterns:
//   - bytes.Index(b, sub) != -1  ->  bytes.Contains(b, sub)
//   - bytes.Index(b, sub) >= 0   ->  bytes.Contains(b, sub)
//   - bytes.Index(b, sub) == -1  ->  !bytes.Contains(b, sub)
//   - bytes.Index(b, sub) < 0    ->  !bytes.Contains(b, sub)
//
// Staticcheck: S1003
type PreferBytesContains struct {
	recipe.Base
}

func (r *PreferBytesContains) Name() string {
	return "org.openrewrite.golang.codequality.PreferBytesContains"
}
func (r *PreferBytesContains) DisplayName() string {
	return "Prefer bytes.Contains over bytes.Index comparison"
}
func (r *PreferBytesContains) Description() string {
	return "Replace `bytes.Index(b, sub) != -1` and similar patterns with `bytes.Contains(b, sub)`."
}
func (r *PreferBytesContains) Tags() []string { return []string{"cleanup", "style"} }

func (r *PreferBytesContains) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{
		{DiagnosticID: "S1003", Tool: diagnostic.Staticcheck, HasFix: true},
	}
}

func (r *PreferBytesContains) RecipeList() []recipe.Recipe {
	return []recipe.Recipe{bytesContainsPositive, bytesContainsNegative}
}
