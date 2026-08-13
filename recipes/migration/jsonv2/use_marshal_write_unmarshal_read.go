/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package jsonv2

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// Rewrites the fluent streaming encode and decode idioms to jsontext-fronted
// MarshalEncode/UnmarshalDecode calls and swaps the import, unless a v1 symbol
// v2 removed would be stranded.
type UseMarshalWriteUnmarshalRead struct {
	recipe.Base
	preserveV1 bool
}

func (r *UseMarshalWriteUnmarshalRead) Name() string {
	return "org.openrewrite.golang.migration.UseMarshalWriteUnmarshalRead"
}
func (r *UseMarshalWriteUnmarshalRead) DisplayName() string {
	return "Migrate streaming `Encode`/`Decode` chains to `jsontext`"
}
func (r *UseMarshalWriteUnmarshalRead) Description() string {
	return "Rewrite `json.NewEncoder(w).Encode(v)` to `json.MarshalEncode(jsontext.NewEncoder(w), v)` and `json.NewDecoder(r).Decode(&v)` to `json.UnmarshalDecode(jsontext.NewDecoder(r), &v)`, swapping the import to `encoding/json/v2`. The jsontext codec preserves v1's streaming contract, including the trailing newline on encode."
}
func (r *UseMarshalWriteUnmarshalRead) Tags() []string { return []string{"migration", "json"} }

func (r *UseMarshalWriteUnmarshalRead) Editor() recipe.TreeVisitor {
	return visitor.Init(&mechanicalMigrator{allowed: mechanicalSet{streaming: true}, preserveV1: r.preserveV1})
}
