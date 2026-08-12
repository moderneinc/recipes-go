/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package jsonv2

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// Rewrites json.MarshalIndent to json.Marshal with jsontext indentation options,
// applied per file only when every encoding/json touchpoint is a MarshalIndent
// call so the whole file stays consistent.
type ReplaceMarshalIndent struct {
	recipe.Base
}

func (r *ReplaceMarshalIndent) Name() string {
	return "org.openrewrite.golang.migration.ReplaceMarshalIndent"
}
func (r *ReplaceMarshalIndent) DisplayName() string {
	return "Replace `json.MarshalIndent`"
}
func (r *ReplaceMarshalIndent) Description() string {
	return "Rewrite `json.MarshalIndent(v, prefix, indent)` to `json.Marshal(v, jsontext.WithIndent(indent), jsontext.WithIndentPrefix(prefix))`, add the `encoding/json/jsontext` import, and swap the `encoding/json` import to `encoding/json/v2`. Applied per file only when every `encoding/json` touchpoint is a `MarshalIndent` call; any other usage leaves the whole file unchanged so it can be migrated and reviewed by other recipes. Dot and blank imports are left untouched."
}
func (r *ReplaceMarshalIndent) Tags() []string { return []string{"migration", "json"} }

func (r *ReplaceMarshalIndent) Editor() recipe.TreeVisitor {
	return visitor.Init(&mechanicalMigrator{allowed: mechanicalSet{marshalIndent: true}})
}
