/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package jsonv2

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
)

// Composes the mechanical encoding/json to encoding/json/v2 rewrites with the
// encoder/decoder relocation, migrating each construct to its idiomatic v2 form
// and adopting v2 semantics.
type MigrateToJSONV2 struct {
	recipe.Base
}

func (r *MigrateToJSONV2) Name() string {
	return "org.openrewrite.golang.migration.MigrateToJSONV2"
}
func (r *MigrateToJSONV2) DisplayName() string {
	return "Migrate `encoding/json` to `encoding/json/v2` (all mechanical rewrites)"
}
func (r *MigrateToJSONV2) Description() string {
	return "Migrate the mechanical `encoding/json` idioms to `encoding/json/v2` by composing the streaming, `MarshalIndent`, and function-local `Encoder`/`Decoder` rewrites, adopting v2 semantics."
}
func (r *MigrateToJSONV2) Tags() []string { return []string{"migration", "json"} }

func (r *MigrateToJSONV2) RecipeList() []recipe.Recipe {
	return []recipe.Recipe{
		&UseMarshalWriteUnmarshalRead{},
		&ReplaceMarshalIndent{},
		&RelocateEncoderDecoderTypes{},
	}
}
