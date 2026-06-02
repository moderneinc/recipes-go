/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package migration_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes-code-quality/recipes/migration"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestChangeGoVersionSetsVersion(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&migration.ChangeGoVersion{NewVersion: "1.22"})
	spec.RewriteRun(t,
		test.GoMod(`
			module example.com/foo

			go 1.21
		`, `
			module example.com/foo

			go 1.22
		`),
	)
}

func TestChangeGoVersionNoChangeWhenAlreadyTarget(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&migration.ChangeGoVersion{NewVersion: "1.22"})
	spec.RewriteRun(t,
		test.GoMod(`
			module example.com/bar

			go 1.22
		`),
	)
}

// ChangeGoVersion is unconditional: unlike UpgradeGoTo126 it will lower the
// version when asked to.
func TestChangeGoVersionDowngrades(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&migration.ChangeGoVersion{NewVersion: "1.21"})
	spec.RewriteRun(t,
		test.GoMod(`
			module example.com/foo

			go 1.25
		`, `
			module example.com/foo

			go 1.21
		`),
	)
}

func TestChangeGoVersionAcrossRequireBlock(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&migration.ChangeGoVersion{NewVersion: "1.23"})
	spec.RewriteRun(t,
		test.GoMod(`
			module example.com/foo

			go 1.21

			require (
				github.com/foo/bar v1.0.0
				github.com/baz/qux v1.5.0 // indirect
			)
		`, `
			module example.com/foo

			go 1.23

			require (
				github.com/foo/bar v1.0.0
				github.com/baz/qux v1.5.0 // indirect
			)
		`),
	)
}
