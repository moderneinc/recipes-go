/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package migration_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes-code-quality/recipes/migration"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestUpgradeGoTo126RaisesOlderVersion(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&migration.UpgradeGoTo126{})
	spec.RewriteRun(t,
		test.GoMod(`
			module example.com/foo

			go 1.21
		`, `
			module example.com/foo

			go 1.26
		`),
	)
}

func TestUpgradeGoTo126LeavesTargetVersion(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&migration.UpgradeGoTo126{})
	spec.RewriteRun(t,
		test.GoMod(`
			module example.com/foo

			go 1.26
		`),
	)
}

func TestUpgradeGoTo126DoesNotDowngrade(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&migration.UpgradeGoTo126{})
	spec.RewriteRun(t,
		test.GoMod(`
			module example.com/foo

			go 1.27
		`),
	)
}

func TestUpgradeGoTo126LeavesGoSourceUntouched(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&migration.UpgradeGoTo126{})
	spec.RewriteRun(t,
		test.GoProject("example",
			test.GoMod(`
				module example.com/foo

				go 1.21
			`, `
				module example.com/foo

				go 1.26
			`),
			test.Golang(`
				package main

				func main() {
					_ = "1.21"
				}
			`),
		),
	)
}

func TestUpgradeGoTo126AcrossRequireBlock(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&migration.UpgradeGoTo126{})
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

			go 1.26

			require (
				github.com/foo/bar v1.0.0
				github.com/baz/qux v1.5.0 // indirect
			)
		`),
	)
}
