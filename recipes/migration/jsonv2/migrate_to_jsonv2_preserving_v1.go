/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package jsonv2

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
)

// Runs the full v2 migration and then preserves v1 semantics, for a
// byte-identical move to the encoding/json/v2 API.
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
	return "Run the full `MigrateToJSONV2` mechanical migration and then `PreserveV1Semantics`, so the code moves to the `encoding/json/v2` API while its output stays byte-identical to v1 through `jsonv1.DefaultOptionsV1()`. Use this instead of `MigrateToJSONV2` when a disruption-free migration is required."
}
func (r *MigrateToJSONV2PreservingV1) Tags() []string { return []string{"migration", "json"} }

func (r *MigrateToJSONV2PreservingV1) RecipeList() []recipe.Recipe {
	return []recipe.Recipe{
		&MigrateToJSONV2{},
		&PreserveV1Semantics{},
	}
}
