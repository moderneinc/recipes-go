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
	wfName = template.Expr("wfName")
	wfData = template.Expr("wfData")
	wfPerm = template.Expr("wfPerm")
)

// PreferOsWriteFile replaces `ioutil.WriteFile(name, data, perm)` with
// `os.WriteFile(name, data, perm)` (Go 1.16+).
// Staticcheck: SA1019 (deprecated)
type PreferOsWriteFile struct {
	recipe.Base
}

func (r *PreferOsWriteFile) Name() string {
	return "org.openrewrite.golang.codequality.PreferOsWriteFile"
}
func (r *PreferOsWriteFile) DisplayName() string {
	return "Prefer os.WriteFile"
}
func (r *PreferOsWriteFile) Description() string {
	return "Replace deprecated `ioutil.WriteFile(name, data, perm)` with `os.WriteFile(name, data, perm)` (Go 1.16+)."
}
func (r *PreferOsWriteFile) Tags() []string { return []string{"cleanup", "simplification"} }

func (r *PreferOsWriteFile) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{
		{DiagnosticID: "SA1019", Tool: diagnostic.Staticcheck, HasFix: true},
	}
}

var preferOsWriteFileImpl = template.NewRecipe(
	template.RecipeName("org.openrewrite.golang.codequality.PreferOsWriteFile$Impl"),
	template.WithDisplayName("ioutil.WriteFile → os.WriteFile"),
	template.WithBefore(fmt.Sprintf(`ioutil.WriteFile(%s, %s, %s)`, wfName, wfData, wfPerm), template.Imports("io/ioutil")),
	template.WithAfter(fmt.Sprintf(`os.WriteFile(%s, %s, %s)`, wfName, wfData, wfPerm), template.Imports("os")),
	template.WithCaptures(wfName, wfData, wfPerm),
)

func (r *PreferOsWriteFile) RecipeList() []recipe.Recipe {
	return []recipe.Recipe{preferOsWriteFileImpl}
}
