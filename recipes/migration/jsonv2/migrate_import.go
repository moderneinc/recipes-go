/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package jsonv2

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// Applies the mechanical encoding/json rewrites together and swaps the import,
// applied per file only when every touchpoint is a mechanical construct so the
// file still compiles.
type MigrateImportToJSONV2 struct {
	recipe.Base
}

func (r *MigrateImportToJSONV2) Name() string {
	return "org.openrewrite.golang.migration.MigrateImportToJSONV2"
}
func (r *MigrateImportToJSONV2) DisplayName() string {
	return "Migrate `encoding/json` to `encoding/json/v2`"
}
func (r *MigrateImportToJSONV2) Description() string {
	return "Rewrite the mechanical `encoding/json` idioms and swap the import to `encoding/json/v2` in one pass: streaming `NewEncoder`/`NewDecoder` chains become `MarshalWrite`/`UnmarshalRead`, and `MarshalIndent` becomes `Marshal` with `jsontext` options, adding `encoding/json/jsontext` when needed. It composes the streaming and `MarshalIndent` rewrites so a file mixing both migrates in one pass, while stored `Encoder`/`Decoder` usage is handled separately by `RelocateEncoderDecoderTypes`. It is applied per file only when every `encoding/json` touchpoint is one of these mechanical constructs; a file with any other touchpoint (a plain `Marshal`/`Unmarshal`, a stored `Encoder`/`Decoder`, an exported type such as `RawMessage` or `Number`, a custom `MarshalJSON`/`UnmarshalJSON`, an `omitempty` tag, or a `[N]byte`/`time.Duration` field) is left unchanged for review. Dot and blank imports are left untouched, and some rewrites are not byte-for-byte behavior preserving."
}
func (r *MigrateImportToJSONV2) Tags() []string { return []string{"migration", "json"} }

func (r *MigrateImportToJSONV2) Editor() recipe.TreeVisitor {
	return visitor.Init(&mechanicalMigrator{allowed: mechanicalSet{
		streaming:     true,
		marshalIndent: true,
	}})
}
