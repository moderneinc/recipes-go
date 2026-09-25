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

// build list: a and b are selected; c/c is absent (not in `go list -m all`).
func usageGraph() ([]golang.GoResolvedDependency, []golang.GoPackageModule) {
	resolved := []golang.GoResolvedDependency{
		{ModulePath: "example.com/app", Main: true},
		{ModulePath: "github.com/a/a", Version: "v1.0.0"},
		{ModulePath: "github.com/b/b", Version: "v1.0.0"},
	}
	pkgs := []golang.GoPackageModule{
		{ImportPath: "fmt", Standard: true},
		{ImportPath: "github.com/a/a", ModulePath: "github.com/a/a", Version: "v1.0.0"},
	}
	return resolved, pkgs
}

func TestRemoveUnusedGoModRequiresDropsAbsentFromBuildList(t *testing.T) {
	// given c/c is absent from the resolved build list, while b/b is an // indirect
	//       require that is present in it (kept even though unreachable)
	spec := test.NewRecipeSpec().WithRecipe(&migration.RemoveUnusedGoModRequires{})
	resolved, pkgs := usageGraph()

	// when / then c/c is removed; b/b is kept because it is in the build list
	spec.RewriteRun(t,
		resolvedGraph(
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
	// given the module absent from the build list is the first block entry
	spec := test.NewRecipeSpec().WithRecipe(&migration.RemoveUnusedGoModRequires{})
	resolved, pkgs := usageGraph()

	// when / then it is removed and the new first entry keeps its own line
	spec.RewriteRun(t,
		resolvedGraph(
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
	// given every entry in the block is absent from the build list
	spec := test.NewRecipeSpec().WithRecipe(&migration.RemoveUnusedGoModRequires{})
	resolved := []golang.GoResolvedDependency{
		{ModulePath: "example.com/app", Main: true},
	}
	pkgs := []golang.GoPackageModule{{ImportPath: "fmt", Standard: true}}

	// when / then the empty block is removed entirely
	spec.RewriteRun(t,
		resolvedGraph(
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
	// given a single-line indirect require absent from the build list, alongside a
	// single-line direct require present in it.
	spec := test.NewRecipeSpec().WithRecipe(&migration.RemoveUnusedGoModRequires{})
	resolved := []golang.GoResolvedDependency{
		{ModulePath: "example.com/app", Main: true},
		{ModulePath: "github.com/a/a", Version: "v1.0.0"},
	}
	pkgs := []golang.GoPackageModule{
		{ImportPath: "github.com/a/a", ModulePath: "github.com/a/a", Version: "v1.0.0"},
	}

	// when / then only the single-line require absent from the build list is dropped
	spec.RewriteRun(t,
		resolvedGraph(
			test.GoMod(`
				module example.com/app

				go 1.22

				require github.com/a/a v1.0.0

				require github.com/c/c v1.0.0 // indirect
			`, `
				module example.com/app

				go 1.22

				require github.com/a/a v1.0.0
			`),
			resolved, pkgs,
		),
	)
}

func TestRemoveUnusedGoModRequiresDropsSelfReference(t *testing.T) {
	// given a /v2 module carrying a stray require on its own v1 major version.
	//       The v1 module provides no imported package; it is "reachable" only
	//       through the main module's own require list, which go mod tidy drops.
	spec := test.NewRecipeSpec().WithRecipe(&migration.RemoveUnusedGoModRequires{})
	resolved := []golang.GoResolvedDependency{
		{ModulePath: "github.com/gocolly/colly/v2", Main: true, Deps: []golang.GoModuleRef{
			{ModulePath: "github.com/gocolly/colly", Version: "v1.2.0"},
			{ModulePath: "github.com/PuerkitoBio/goquery", Version: "v1.11.0"},
		}},
		{ModulePath: "github.com/gocolly/colly", Version: "v1.2.0"},
		{ModulePath: "github.com/PuerkitoBio/goquery", Version: "v1.11.0"},
	}
	pkgs := []golang.GoPackageModule{
		{ImportPath: "github.com/gocolly/colly/v2", ModulePath: "github.com/gocolly/colly/v2"},
		{ImportPath: "github.com/PuerkitoBio/goquery", ModulePath: "github.com/PuerkitoBio/goquery", Version: "v1.11.0"},
	}

	// when / then the v1 self-reference is removed; the imported goquery is kept
	spec.RewriteRun(t,
		resolvedGraph(
			test.GoMod(`
				module github.com/gocolly/colly/v2

				go 1.24

				require (
					github.com/PuerkitoBio/goquery v1.11.0
					github.com/gocolly/colly v1.2.0
				)
			`, `
				module github.com/gocolly/colly/v2

				go 1.24

				require (
					github.com/PuerkitoBio/goquery v1.11.0
				)
			`),
			resolved, pkgs,
		),
	)
}

func TestRemoveUnusedGoModRequiresKeepsBuildConstraintDirect(t *testing.T) {
	// given M is a direct require whose only import lives behind a build
	//       constraint not active when the LST was built, so M is absent from
	//       PackageModules; M pulls in the indirect deps i1/i2. `go mod tidy`
	//       keeps all three, so the recipe must not delete a direct require it
	//       merely failed to observe imported, nor cascade to its closure.
	spec := test.NewRecipeSpec().WithRecipe(&migration.RemoveUnusedGoModRequires{})
	resolved := []golang.GoResolvedDependency{
		{ModulePath: "example.com/app", Main: true, Deps: []golang.GoModuleRef{
			{ModulePath: "github.com/gated/m", Version: "v1.9.0"},
		}},
		{ModulePath: "github.com/gated/m", Version: "v1.9.0", Deps: []golang.GoModuleRef{
			{ModulePath: "github.com/i/one", Version: "v1.0.0"},
			{ModulePath: "github.com/i/two", Version: "v1.0.0"},
		}},
		{ModulePath: "github.com/i/one", Version: "v1.0.0", Indirect: true},
		{ModulePath: "github.com/i/two", Version: "v1.0.0", Indirect: true},
	}
	pkgs := []golang.GoPackageModule{{ImportPath: "fmt", Standard: true}}

	// when / then M and its indirect closure are retained
	spec.RewriteRun(t,
		resolvedGraph(
			test.GoMod(`
				module example.com/app

				go 1.22

				require github.com/gated/m v1.9.0

				require (
					github.com/i/one v1.0.0 // indirect
					github.com/i/two v1.0.0 // indirect
				)
			`),
			resolved, pkgs,
		),
	)
}

func TestRemoveUnusedGoModRequiresKeepsUnreachableBuildListIndirect(t *testing.T) {
	// given an // indirect require whose module is in the resolved build list
	//       (go list -m all) but is neither imported nor reachable through the
	//       recorded `go mod graph` edges — the shape of a test-closure or
	//       build-tag-gated transitive dep whose justifying edge is pruned out of
	//       the offline graph (e.g. gotest.tools/v3, onsi/ginkgo, otel/sdk in the
	//       real sweep). `go mod tidy` keeps it, so the recipe must not remove it.
	spec := test.NewRecipeSpec().WithRecipe(&migration.RemoveUnusedGoModRequires{})
	resolved := []golang.GoResolvedDependency{
		{ModulePath: "example.com/app", Main: true, Deps: []golang.GoModuleRef{
			{ModulePath: "github.com/a/a", Version: "v1.0.0"},
		}},
		{ModulePath: "github.com/a/a", Version: "v1.0.0"},
		{ModulePath: "gotest.tools/v3", Version: "v3.5.0", Indirect: true},
	}
	pkgs := []golang.GoPackageModule{
		{ImportPath: "github.com/a/a", ModulePath: "github.com/a/a", Version: "v1.0.0"},
	}

	// when / then the unreachable build-list indirect is retained (no change)
	spec.RewriteRun(t,
		resolvedGraph(
			test.GoMod(`
				module example.com/app

				go 1.22

				require github.com/a/a v1.0.0

				require gotest.tools/v3 v3.5.0 // indirect
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
