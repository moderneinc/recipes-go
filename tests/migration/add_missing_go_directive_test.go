/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package migration_test

import (
	"fmt"
	"testing"

	"github.com/moderneinc/recipes-go/recipes/migration"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestAddMissingGoDirectiveInsertsToolchainVersionAfterModule(t *testing.T) {
	// given a go.mod with a module directive and a require, but no `go` directive
	spec := test.NewRecipeSpec().WithRecipe(&migration.AddMissingGoDirective{})

	// when / then the toolchain version is inserted right after the module line
	v := migration.DefaultGoDirectiveVersion()
	spec.RewriteRun(t,
		test.GoMod(
			`
			module example.com/app

			require github.com/foo/bar v1.0.0
		`,
			fmt.Sprintf(`
			module example.com/app

			go %s

			require github.com/foo/bar v1.0.0
		`, v)),
	)
}

func TestAddMissingGoDirectiveInsertsIntoMinimalModule(t *testing.T) {
	// given a bare go.mod with only a module directive
	spec := test.NewRecipeSpec().WithRecipe(&migration.AddMissingGoDirective{})

	// when / then the `go` directive is appended
	v := migration.DefaultGoDirectiveVersion()
	spec.RewriteRun(t,
		test.GoMod(
			`
			module example.com/app
		`,
			fmt.Sprintf(`
			module example.com/app

			go %s
		`, v)),
	)
}

func TestAddMissingGoDirectiveHonorsConfiguredVersion(t *testing.T) {
	// given a configured version
	spec := test.NewRecipeSpec().WithRecipe(&migration.AddMissingGoDirective{GoVersion: "1.24"})

	// when / then that version is used, not the toolchain default
	spec.RewriteRun(t,
		test.GoMod(`
			module example.com/app
		`, `
			module example.com/app

			go 1.24
		`),
	)
}

func TestAddMissingGoDirectiveNoOpWhenPresent(t *testing.T) {
	// given a go.mod that already carries a `go` directive
	spec := test.NewRecipeSpec().WithRecipe(&migration.AddMissingGoDirective{})

	// when / then nothing changes, including the existing version
	spec.RewriteRun(t,
		test.GoMod(`
			module example.com/app

			go 1.22
		`),
	)
}
