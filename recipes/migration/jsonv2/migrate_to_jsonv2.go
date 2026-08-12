/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package jsonv2

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
)

// Composes the mechanical encoding/json to encoding/json/v2 rewrites, running
// the import, streaming, and MarshalIndent migration together with the
// encoder/decoder relocation as a single recipe.
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
	return "Apply all mechanical `encoding/json` to `encoding/json/v2` rewrites as one recipe by composing `MigrateImportToJSONV2` (streaming `Encode`/`Decode` chains and `MarshalIndent`, with the import swap) and `RelocateEncoderDecoderTypes` (function-local `Encoder`/`Decoder` values). Each sub-recipe keeps its own per-file safety check, so a file is migrated only when all of its `encoding/json` usage is mechanically handled; changed-semantics touchpoints such as a plain `Marshal`/`Unmarshal`, `RawMessage`, or an `omitempty` tag are left for review."
}
func (r *MigrateToJSONV2) Tags() []string { return []string{"migration", "json"} }

func (r *MigrateToJSONV2) RecipeList() []recipe.Recipe {
	return []recipe.Recipe{
		&MigrateImportToJSONV2{},
		&RelocateEncoderDecoderTypes{},
	}
}
