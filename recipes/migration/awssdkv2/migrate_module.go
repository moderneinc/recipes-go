/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package awssdkv2

import (
	"github.com/moderneinc/recipes-go/recipes/migration"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
)

// The entry point: the source migration plus the go.mod that goes with it.
type MigrateAwsSdkGoModuleToV2 struct {
	recipe.Base
}

func (r *MigrateAwsSdkGoModuleToV2) Name() string {
	return "org.openrewrite.golang.migration.MigrateAwsSdkGoModuleToV2"
}
func (r *MigrateAwsSdkGoModuleToV2) DisplayName() string {
	return "Migrate a module from `aws-sdk-go` to `aws-sdk-go-v2`"
}
func (r *MigrateAwsSdkGoModuleToV2) Description() string {
	return "Migrate the files whose `github.com/aws/aws-sdk-go` usage has a faithful `aws-sdk-go-v2` form, and bring go.mod along with them. This is a partial migration by design: v2 replaced the session, the error types, the page iterators and the waiters outright, so a file holding one of those is left as it is. Run `FindAwsSdkGoV1Usage` to enumerate what remains, and `go mod tidy` to resolve the per-service modules."
}
func (r *MigrateAwsSdkGoModuleToV2) Tags() []string {
	return []string{"migration", "aws"}
}

func (r *MigrateAwsSdkGoModuleToV2) RecipeList() []recipe.Recipe {
	return []recipe.Recipe{
		// One scanning sub-recipe only: the Moderne CLI hangs on a recipe list
		// holding more than one, and the source migration is the scanning one
		// because it generates the slice helpers.
		&MigrateAwsSdkGoToV2{},
		&migration.FormatGoMod{},
	}
}
