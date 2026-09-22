/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package mapstructurev2

import (
	"github.com/moderneinc/recipes-go/recipes/migration/internal/depswap"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
)

var mapstructureModules = depswap.Modules{Old: oldModule, New: []string{newModule}, Version: newVersion}

// Brings the go.mod require into line with what the source imports.
type UpdateMapstructureDependency struct {
	recipe.ScanningBase
}

func (r *UpdateMapstructureDependency) Name() string {
	return "org.openrewrite.golang.migration.UpdateMapstructureDependency"
}
func (r *UpdateMapstructureDependency) DisplayName() string {
	return "Require `go-viper/mapstructure/v2` instead of `mitchellh/mapstructure`"
}
func (r *UpdateMapstructureDependency) Description() string {
	return "Bring the go.mod `require` into line with what the source imports: repoint `github.com/mitchellh/mapstructure` at `github.com/go-viper/mapstructure/v2 " + newVersion + "` once nothing imports the unmaintained module any more, and require both while a file naming the dropped `mapstructure.Error` still does. Does not sync go.sum, so a `go mod tidy` is still needed to complete resolution."
}
func (r *UpdateMapstructureDependency) Tags() []string {
	return []string{"migration", "mapstructure", "gomod"}
}

func (r *UpdateMapstructureDependency) InitialValue(*recipe.ExecutionContext) any {
	return depswap.NewAcc()
}
func (r *UpdateMapstructureDependency) Scanner(acc any) recipe.TreeVisitor {
	return depswap.Scanner(acc.(*depswap.Acc), mapstructureModules,
		depswap.EditorProbe((&SwapMapstructureImports{}).Editor))
}
func (r *UpdateMapstructureDependency) EditorWithData(acc any) recipe.TreeVisitor {
	return depswap.Editor(acc.(*depswap.Acc), mapstructureModules)
}
