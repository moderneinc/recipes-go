/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package jsonv2

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
)

// Composes MigrateToJSONV2 and PreserveV1Semantics for a low-disruption move to
// encoding/json/v2 that preserves v1 behavior.
type MigrateToJSONV2PreservingV1 struct {
	recipe.Base
}

func (r *MigrateToJSONV2PreservingV1) Name() string {
	return "org.openrewrite.golang.migration.MigrateToJSONV2PreservingV1"
}
func (r *MigrateToJSONV2PreservingV1) DisplayName() string {
	return "Migrate `encoding/json` to `encoding/json/v2`, preserving v1 semantics"
}
func (r *MigrateToJSONV2PreservingV1) Description() string {
	return "Migrate `encoding/json` to `encoding/json/v2` while preserving v1 behavior, by composing `MigrateToJSONV2` and `PreserveV1Semantics`. Use it instead of `MigrateToJSONV2` for a low-disruption migration."
}
func (r *MigrateToJSONV2PreservingV1) Tags() []string { return []string{"migration", "json"} }

func (r *MigrateToJSONV2PreservingV1) RecipeList() []recipe.Recipe {
	return []recipe.Recipe{
		&MigrateToJSONV2{},
		&PreserveV1Semantics{},
	}
}
