/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package migration_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes/migration"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestUpgradeGoToRaisesOlderVersion(t *testing.T) {
	cases := []struct {
		recipe recipe.Recipe
		target string
	}{
		{&migration.UpgradeGoTo118{}, "1.18"},
		{&migration.UpgradeGoTo119{}, "1.19"},
		{&migration.UpgradeGoTo120{}, "1.20"},
		{&migration.UpgradeGoTo121{}, "1.21"},
		{&migration.UpgradeGoTo122{}, "1.22"},
		{&migration.UpgradeGoTo123{}, "1.23"},
		{&migration.UpgradeGoTo124{}, "1.24"},
		{&migration.UpgradeGoTo125{}, "1.25"},
		{&migration.UpgradeGoTo126{}, "1.26"},
	}
	for _, c := range cases {
		t.Run(c.target, func(t *testing.T) {
			spec := test.NewRecipeSpec().WithRecipe(c.recipe)
			spec.RewriteRun(t,
				test.GoMod(`
					module example.com/foo

					go 1.16
				`, `
					module example.com/foo

					go `+c.target+`
				`),
			)
		})
	}
}

func TestUpgradeGoTo125LeavesTargetVersion(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&migration.UpgradeGoTo125{})
	spec.RewriteRun(t,
		test.GoMod(`
			module example.com/foo

			go 1.25
		`),
	)
}

func TestUpgradeGoTo123DoesNotDowngrade(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&migration.UpgradeGoTo123{})
	spec.RewriteRun(t,
		test.GoMod(`
			module example.com/foo

			go 1.25
		`),
	)
}

func TestUpgradeGoTo126LeavesToolchainAndGoSourceUntouched(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&migration.UpgradeGoTo126{})
	spec.RewriteRun(t,
		test.GoProject("example",
			test.GoMod(`
				module example.com/foo

				go 1.21

				toolchain go1.21.5
			`, `
				module example.com/foo

				go 1.26

				toolchain go1.21.5
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
