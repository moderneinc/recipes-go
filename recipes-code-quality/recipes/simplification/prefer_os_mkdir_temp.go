/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification

import (
	"fmt"

	"github.com/moderneinc/recipes-go/code-quality/diagnostic"
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/template"
)

var (
	tdDir = template.Expr("tdDir")
	tdPat = template.Expr("tdPat")
)

// PreferOsMkdirTemp replaces `ioutil.TempDir(dir, pattern)` with
// `os.MkdirTemp(dir, pattern)` (Go 1.16+).
// Staticcheck: SA1019 (deprecated)
type PreferOsMkdirTemp struct {
	recipe.Base
}

func (r *PreferOsMkdirTemp) Name() string {
	return "org.openrewrite.golang.codequality.PreferOsMkdirTemp"
}
func (r *PreferOsMkdirTemp) DisplayName() string {
	return "Prefer os.MkdirTemp"
}
func (r *PreferOsMkdirTemp) Description() string {
	return "Replace deprecated `ioutil.TempDir(dir, pattern)` with `os.MkdirTemp(dir, pattern)` (Go 1.16+)."
}
func (r *PreferOsMkdirTemp) Tags() []string { return []string{"cleanup", "simplification"} }

func (r *PreferOsMkdirTemp) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{
		{DiagnosticID: "SA1019", Tool: diagnostic.Staticcheck, HasFix: true},
	}
}

var preferOsMkdirTempImpl = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.PreferOsMkdirTemp$Impl"),
	template.WithDisplayName("ioutil.TempDir → os.MkdirTemp"),
	template.WithBefore(fmt.Sprintf(`ioutil.TempDir(%s, %s)`, tdDir, tdPat), template.Imports("io/ioutil")),
	template.WithAfter(fmt.Sprintf(`os.MkdirTemp(%s, %s)`, tdDir, tdPat), template.Imports("os")),
	template.WithCaptures(tdDir, tdPat),
)

func (r *PreferOsMkdirTemp) RecipeList() []recipe.Recipe {
	return []recipe.Recipe{preferOsMkdirTempImpl}
}
