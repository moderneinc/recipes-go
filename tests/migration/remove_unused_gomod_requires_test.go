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

// graph: a is imported; a -> b (so b is reachable); c is dead.
func usageGraph() ([]golang.GoResolvedDependency, []golang.GoPackageModule) {
	resolved := []golang.GoResolvedDependency{
		{ModulePath: "example.com/app", Main: true},
		{ModulePath: "github.com/a/a", Version: "v1.0.0", Deps: []golang.GoModuleRef{{ModulePath: "github.com/b/b", Version: "v1.0.0"}}},
		{ModulePath: "github.com/b/b", Version: "v1.0.0"},
		{ModulePath: "github.com/c/c", Version: "v1.0.0"},
	}
	pkgs := []golang.GoPackageModule{
		{ImportPath: "fmt", Standard: true},
		{ImportPath: "github.com/a/a", ModulePath: "github.com/a/a", Version: "v1.0.0"},
	}
	return resolved, pkgs
}

func TestRemoveUnusedGoModRequiresDropsUnreachable(t *testing.T) {
	// given c/c is neither imported nor reachable from an imported module
	spec := test.NewRecipeSpec().WithRecipe(&migration.RemoveUnusedGoModRequires{})
	resolved, pkgs := usageGraph()

	// when / then c/c is removed; b/b is kept because a/a (imported) requires it
	spec.RewriteRun(t,
		test.GoModGraph(
			test.GoMod(`
				module example.com/app

				go 1.22

				require (
					github.com/a/a v1.0.0
					github.com/b/b v1.0.0 // indirect
					github.com/c/c v1.0.0 // indirect
				)
			`, `
				module example.com/app

				go 1.22

				require (
					github.com/a/a v1.0.0
					github.com/b/b v1.0.0 // indirect
				)
			`),
			resolved, pkgs,
		),
	)
}

func TestRemoveUnusedGoModRequiresFixesFirstEntryDrop(t *testing.T) {
	// given the unreachable module is the first block entry
	spec := test.NewRecipeSpec().WithRecipe(&migration.RemoveUnusedGoModRequires{})
	resolved, pkgs := usageGraph()

	// when / then it is removed and the new first entry keeps its own line
	spec.RewriteRun(t,
		test.GoModGraph(
			test.GoMod(`
				module example.com/app

				go 1.22

				require (
					github.com/c/c v1.0.0 // indirect
					github.com/a/a v1.0.0
					github.com/b/b v1.0.0 // indirect
				)
			`, `
				module example.com/app

				go 1.22

				require (
					github.com/a/a v1.0.0
					github.com/b/b v1.0.0 // indirect
				)
			`),
			resolved, pkgs,
		),
	)
}

func TestRemoveUnusedGoModRequiresDropsWholeBlock(t *testing.T) {
	// given every entry in the block is unused
	spec := test.NewRecipeSpec().WithRecipe(&migration.RemoveUnusedGoModRequires{})
	resolved := []golang.GoResolvedDependency{
		{ModulePath: "example.com/app", Main: true},
		{ModulePath: "github.com/c/c", Version: "v1.0.0"},
	}
	pkgs := []golang.GoPackageModule{{ImportPath: "fmt", Standard: true}}

	// when / then the empty block is removed entirely
	spec.RewriteRun(t,
		test.GoModGraph(
			test.GoMod(`
				module example.com/app

				go 1.22

				require (
					github.com/c/c v1.0.0 // indirect
				)
			`, `
				module example.com/app

				go 1.22
			`),
			resolved, pkgs,
		),
	)
}

func TestRemoveUnusedGoModRequiresSingleLine(t *testing.T) {
	// given an unused single-line require
	spec := test.NewRecipeSpec().WithRecipe(&migration.RemoveUnusedGoModRequires{})
	resolved := []golang.GoResolvedDependency{
		{ModulePath: "example.com/app", Main: true},
		{ModulePath: "github.com/a/a", Version: "v1.0.0"},
		{ModulePath: "github.com/c/c", Version: "v1.0.0"},
	}
	pkgs := []golang.GoPackageModule{
		{ImportPath: "github.com/a/a", ModulePath: "github.com/a/a", Version: "v1.0.0"},
	}

	// when / then only the unused single-line require is dropped
	spec.RewriteRun(t,
		test.GoModGraph(
			test.GoMod(`
				module example.com/app

				go 1.22

				require github.com/a/a v1.0.0

				require github.com/c/c v1.0.0
			`, `
				module example.com/app

				go 1.22

				require github.com/a/a v1.0.0
			`),
			resolved, pkgs,
		),
	)
}

func TestRemoveUnusedGoModRequiresNoChangeWithoutResolution(t *testing.T) {
	// given no resolved package→module map (resolution did not run)
	spec := test.NewRecipeSpec().WithRecipe(&migration.RemoveUnusedGoModRequires{})

	// when / then no change — the recipe must not remove anything blindly
	spec.RewriteRun(t,
		test.GoMod(`
			module example.com/app

			go 1.22

			require (
				github.com/a/a v1.0.0
				github.com/c/c v1.0.0 // indirect
			)
		`),
	)
}
