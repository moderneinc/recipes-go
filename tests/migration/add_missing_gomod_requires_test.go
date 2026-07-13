/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package migration_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes/migration"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
)

func TestAddMissingGoModRequiresAppendsToBlock(t *testing.T) {
	// given a resolved build list with two modules go.mod does not declare
	spec := test.NewRecipeSpec().WithRecipe(&migration.AddMissingGoModRequires{})
	resolved := []golang.GoResolvedDependency{
		{ModulePath: "example.com/app", Main: true},
		{ModulePath: "github.com/foo/bar", Version: "v1.2.3"},
		{ModulePath: "github.com/baz/qux", Version: "v1.0.0"},
		{ModulePath: "golang.org/x/text", Version: "v0.3.0", Indirect: true},
	}

	// when / then the missing modules are appended, indirect flag preserved
	spec.RewriteRun(t,
		test.GoModGraph(
			test.GoMod(`
				module example.com/app

				go 1.22

				require (
					github.com/foo/bar v1.2.3
				)
			`, `
				module example.com/app

				go 1.22

				require (
					github.com/foo/bar v1.2.3
					github.com/baz/qux v1.0.0
					golang.org/x/text v0.3.0 // indirect
				)
			`),
			resolved, nil,
		),
	)
}

func TestAddMissingGoModRequiresCreatesBlockWhenNone(t *testing.T) {
	// given only a single-line require and a missing module
	spec := test.NewRecipeSpec().WithRecipe(&migration.AddMissingGoModRequires{})
	resolved := []golang.GoResolvedDependency{
		{ModulePath: "example.com/app", Main: true},
		{ModulePath: "github.com/foo/bar", Version: "v1.2.3"},
		{ModulePath: "github.com/baz/qux", Version: "v1.0.0"},
	}

	// when / then a new require block is created for the missing module
	spec.RewriteRun(t,
		test.GoModGraph(
			test.GoMod(`
				module example.com/app

				go 1.22

				require github.com/foo/bar v1.2.3
			`, `
				module example.com/app

				go 1.22

				require github.com/foo/bar v1.2.3

				require (
					github.com/baz/qux v1.0.0
				)
			`),
			resolved, nil,
		),
	)
}

func TestAddMissingGoModRequiresNoChangeWhenAllDeclared(t *testing.T) {
	// given a build list fully covered by go.mod
	spec := test.NewRecipeSpec().WithRecipe(&migration.AddMissingGoModRequires{})
	resolved := []golang.GoResolvedDependency{
		{ModulePath: "example.com/app", Main: true},
		{ModulePath: "github.com/foo/bar", Version: "v1.2.3"},
	}

	// when / then no change
	spec.RewriteRun(t,
		test.GoModGraph(
			test.GoMod(`
				module example.com/app

				go 1.22

				require github.com/foo/bar v1.2.3
			`),
			resolved, nil,
		),
	)
}

func TestAddMissingGoModRequiresNoChangeWithoutResolution(t *testing.T) {
	// given no resolved build list (parse-time resolution gate was off)
	spec := test.NewRecipeSpec().WithRecipe(&migration.AddMissingGoModRequires{})

	// when / then no change — the recipe degrades to a no-op
	spec.RewriteRun(t,
		test.GoMod(`
			module example.com/app

			go 1.22

			require github.com/foo/bar v1.2.3
		`),
	)
}
