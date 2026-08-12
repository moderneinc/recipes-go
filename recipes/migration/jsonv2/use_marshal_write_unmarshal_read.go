/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package jsonv2

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// Rewrites the fluent streaming encode and decode idioms to their v2 package
// functions, applied per file only when every encoding/json touchpoint is such
// a chain so the whole file stays consistent.
type UseMarshalWriteUnmarshalRead struct {
	recipe.Base
}

func (r *UseMarshalWriteUnmarshalRead) Name() string {
	return "org.openrewrite.golang.migration.UseMarshalWriteUnmarshalRead"
}
func (r *UseMarshalWriteUnmarshalRead) DisplayName() string {
	return "Use `json.MarshalWrite` and `json.UnmarshalRead`"
}
func (r *UseMarshalWriteUnmarshalRead) Description() string {
	return "Rewrite the fluent streaming idioms `json.NewEncoder(w).Encode(v)` to `json.MarshalWrite(w, v)` and `json.NewDecoder(r).Decode(&v)` to `json.UnmarshalRead(r, &v)`, and swap the `encoding/json` import to `encoding/json/v2`. The rewrite is applied per file only when every `encoding/json` touchpoint is one of these chains; any other touchpoint (a package function such as `Marshal` or `MarshalIndent`, a stored `Encoder`/`Decoder`, an exported type such as `RawMessage` or `Number`, a custom `MarshalJSON`/`UnmarshalJSON`, an `omitempty` tag, or a `[N]byte`/`time.Duration` field) leaves the whole file unchanged so it can be migrated and reviewed by other recipes. Dot and blank imports are left untouched. The v2 targets are not byte-for-byte behavior preserving: `MarshalWrite` does not append the trailing newline that `Encoder.Encode` wrote, and `UnmarshalRead` consumes the reader to EOF rather than reading a single value, so newline-delimited and multi-value streaming is affected."
}
func (r *UseMarshalWriteUnmarshalRead) Tags() []string { return []string{"migration", "json"} }

func (r *UseMarshalWriteUnmarshalRead) Editor() recipe.TreeVisitor {
	return visitor.Init(&mechanicalMigrator{allowed: mechanicalSet{streaming: true}})
}
