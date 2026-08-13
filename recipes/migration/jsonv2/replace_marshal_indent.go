/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package jsonv2

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// Rewrites json.MarshalIndent to json.Marshal with jsontext indentation options
// and swaps the import, unless a v1 symbol v2 removed would be stranded.
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
	return "Rewrite `json.MarshalIndent(v, prefix, indent)` to `json.Marshal(v, jsontext.WithIndent(indent), jsontext.WithIndentPrefix(prefix))`, swapping the import to `encoding/json/v2`."
}
func (r *ReplaceMarshalIndent) Tags() []string { return []string{"migration", "json"} }

func (r *ReplaceMarshalIndent) Editor() recipe.TreeVisitor {
	return visitor.Init(&mechanicalMigrator{allowed: mechanicalSet{marshalIndent: true}})
}
