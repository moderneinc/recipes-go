/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package ubermock

import (
	"github.com/moderneinc/recipes-go/diagnostic"
	"github.com/moderneinc/recipes-go/recipes/migration"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
)

// The entry point: the whole source and go.mod migration off the archived mock
// library.
type MigrateToUberMock struct {
	recipe.Base
}

func (r *MigrateToUberMock) Name() string {
	return "org.openrewrite.golang.migration.MigrateToUberMock"
}
func (r *MigrateToUberMock) DisplayName() string {
	return "Migrate from `github.com/golang/mock` to `go.uber.org/mock`"
}
func (r *MigrateToUberMock) Description() string {
	return "Migrate off `github.com/golang/mock`, which Google archived in June 2023, to the maintained fork `go.uber.org/mock`. The fork's `gomock` API is a superset of the original's, so this is a path swap: imports, `//go:generate mockgen` directives and the go.mod `require` all move, and call sites are untouched. Run `RemoveRedundantGomockFinish` afterwards to drop the `defer ctrl.Finish()` calls the fork makes unnecessary, and `go mod tidy` to sync go.sum."
}
func (r *MigrateToUberMock) Tags() []string { return []string{"migration", "mock", "testing"} }

func (r *MigrateToUberMock) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{
		{DiagnosticID: "SA1019", Tool: diagnostic.Staticcheck, HasFix: true},
	}
}

func (r *MigrateToUberMock) RecipeList() []recipe.Recipe {
	return []recipe.Recipe{
		&SwapGolangMockImports{},
		&UpdateMockgenGoGenerateDirectives{},
		&UpdateGolangMockDependency{},
		// The repointed require sits where the old module path sorted.
		&migration.FormatGoMod{},
	}
}
