/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package mapstructurev2

import (
	"github.com/moderneinc/recipes-go/recipes/migration"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
)

// The entry point: the whole source and go.mod migration to the maintained fork.
type MigrateToGoViperMapstructure struct {
	recipe.Base
}

func (r *MigrateToGoViperMapstructure) Name() string {
	return "org.openrewrite.golang.migration.MigrateToGoViperMapstructure"
}
func (r *MigrateToGoViperMapstructure) DisplayName() string {
	return "Migrate from `mitchellh/mapstructure` to `go-viper/mapstructure/v2`"
}
func (r *MigrateToGoViperMapstructure) Description() string {
	return "Migrate off `github.com/mitchellh/mapstructure`, unmaintained since 2023, to the community fork `github.com/go-viper/mapstructure/v2`. v2 keeps every function, hook and `DecoderConfig` field of v1 and adds more, so this is a path swap. The one exception is the exported `Error` struct, which v2 replaced with `errors.Join`; a file naming it is left untouched and reported by `FindMapstructureErrorUsage`. Run `go mod tidy` afterwards to sync go.sum."
}
func (r *MigrateToGoViperMapstructure) Tags() []string {
	return []string{"migration", "mapstructure"}
}

func (r *MigrateToGoViperMapstructure) RecipeList() []recipe.Recipe {
	return []recipe.Recipe{
		&SwapMapstructureImports{},
		&UpdateMapstructureDependency{},
		// The repointed require sits where the old module path sorted.
		&migration.FormatGoMod{},
	}
}
