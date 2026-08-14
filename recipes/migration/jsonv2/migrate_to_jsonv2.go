/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package jsonv2

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
)

// Composes the mechanical encoding/json to encoding/json/v2 rewrites, migrating
// each construct to its idiomatic v2 form and adopting v2 semantics.
type MigrateToJSONV2 struct {
	recipe.Base
	preserveV1 bool
}

func (r *MigrateToJSONV2) Name() string {
	return "org.openrewrite.golang.migration.MigrateToJSONV2"
}
func (r *MigrateToJSONV2) DisplayName() string {
	return "Migrate `encoding/json` to `encoding/json/v2` (all mechanical rewrites)"
}
func (r *MigrateToJSONV2) Description() string {
	return "Migrate the mechanical `encoding/json` idioms to `encoding/json/v2` by composing the streaming, `MarshalIndent`, function-local `Encoder`/`Decoder`, and `RawMessage` rewrites plus an import-only swap for files whose usage already exists in v2, adopting v2 semantics. To keep v1 output byte-identical instead, run the opt-in `PreserveV1Semantics` recipe afterwards."
}
func (r *MigrateToJSONV2) Tags() []string { return []string{"migration", "json"} }

func (r *MigrateToJSONV2) RecipeList() []recipe.Recipe {
	return []recipe.Recipe{
		&MigrateStreamingEncodeDecode{preserveV1: r.preserveV1},
		&ReplaceMarshalIndent{preserveV1: r.preserveV1},
		&RelocateEncoderDecoderTypes{preserveV1: r.preserveV1},
		&RelocateRawMessage{preserveV1: r.preserveV1},
		&MigrateImportOnlyToJSONV2{preserveV1: r.preserveV1},
	}
}
