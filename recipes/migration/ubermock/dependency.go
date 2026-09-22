/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package ubermock

import (
	"github.com/moderneinc/recipes-go/recipes/migration/internal/depswap"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
)

var mockModules = depswap.Modules{Old: oldModule, New: []string{newModule}, Version: newVersion}

// Brings the go.mod require into line with what the source imports.
type UpdateGolangMockDependency struct {
	recipe.ScanningBase
}

func (r *UpdateGolangMockDependency) Name() string {
	return "org.openrewrite.golang.migration.UpdateGolangMockDependency"
}
func (r *UpdateGolangMockDependency) DisplayName() string {
	return "Require `go.uber.org/mock` instead of `github.com/golang/mock`"
}
func (r *UpdateGolangMockDependency) Description() string {
	return "Bring the go.mod `require` into line with what the source imports: repoint `github.com/golang/mock` at `go.uber.org/mock " + newVersion + "` once nothing imports the archived module any more, and require both while a file still does. Does not sync go.sum, so a `go mod tidy` is still needed to complete resolution."
}
func (r *UpdateGolangMockDependency) Tags() []string {
	return []string{"migration", "mock", "gomod"}
}

func (r *UpdateGolangMockDependency) InitialValue(*recipe.ExecutionContext) any {
	return depswap.NewAcc()
}
func (r *UpdateGolangMockDependency) Scanner(acc any) recipe.TreeVisitor {
	return depswap.Scanner(acc.(*depswap.Acc), mockModules,
		depswap.EditorProbe((&SwapGolangMockImports{}).Editor))
}
func (r *UpdateGolangMockDependency) EditorWithData(acc any) recipe.TreeVisitor {
	return depswap.Editor(acc.(*depswap.Acc), mockModules)
}
