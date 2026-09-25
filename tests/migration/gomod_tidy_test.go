/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package migration_test

import (
	"fmt"
	"testing"

	"github.com/moderneinc/recipes-go/recipes/migration"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
)

// TestGoModTidyComposite exercises the whole pipeline end to end: a go.mod that
// is missing a build-list module, carries an unused one, and is unsorted, run
// through the composite with the module graph injected and one .go source.
func TestGoModTidyComposite(t *testing.T) {
	// given foo/bar is imported; foo/bar -> baz/qux (transitive); dead/mod is unreachable;
	// baz/qux is in the build list but absent from go.mod.
	spec := test.NewRecipeSpec().WithRecipe(&migration.GoModTidy{})
	resolved := []golang.GoResolvedDependency{
		{ModulePath: "example.com/app", Main: true},
		{ModulePath: "github.com/foo/bar", Version: "v1.0.0", Deps: []golang.GoModuleRef{{ModulePath: "github.com/baz/qux", Version: "v1.0.0"}}},
		{ModulePath: "github.com/baz/qux", Version: "v1.0.0", Indirect: true},
		{ModulePath: "github.com/dead/mod", Version: "v1.0.0", Indirect: true},
	}
	pkgs := []golang.GoPackageModule{
		{ImportPath: "github.com/foo/bar", ModulePath: "github.com/foo/bar", Version: "v1.0.0"},
		{ImportPath: "github.com/baz/qux", ModulePath: "github.com/baz/qux", Version: "v1.0.0"},
	}

	// when / then: dead/mod removed, baz/qux added to its own indirect block, blocks sorted.
	spec.RewriteRun(t,
		test.GoProject("app",
			resolvedGraph(
				test.GoMod(`
					module example.com/app

					go 1.22

					require (
						github.com/foo/bar v1.0.0
						github.com/dead/mod v1.0.0 // indirect
					)
				`, `
					module example.com/app

					go 1.22

					require (
						github.com/foo/bar v1.0.0
					)

					require (
						github.com/baz/qux v1.0.0 // indirect
					)
				`),
				resolved, pkgs,
			),
			test.Golang(`
				package main

				import "github.com/foo/bar"

				func main() { _ = bar.A }
			`),
		),
	)
}

func TestGoModTidyAddsMissingGoDirective(t *testing.T) {
	// given a resolved graph where foo/bar is imported and already required, but
	// the go.mod carries no `go` directive.
	spec := test.NewRecipeSpec().WithRecipe(&migration.GoModTidy{})
	resolved := []golang.GoResolvedDependency{
		{ModulePath: "example.com/app", Main: true},
		{ModulePath: "github.com/foo/bar", Version: "v1.0.0"},
	}
	pkgs := []golang.GoPackageModule{
		{ImportPath: "github.com/foo/bar", ModulePath: "github.com/foo/bar", Version: "v1.0.0"},
	}

	// when / then a `go` directive should be added, as `go mod tidy` does.
	spec.RewriteRun(t,
		test.GoProject("app",
			resolvedGraph(
				test.GoMod(
					`
					module example.com/app

					require github.com/foo/bar v1.0.0
				`,
					fmt.Sprintf(`
					module example.com/app

					go %s

					require github.com/foo/bar v1.0.0
				`, migration.DefaultGoDirectiveVersion())),
				resolved, pkgs,
			),
			test.Golang(`
				package main

				import "github.com/foo/bar"

				func main() { _ = bar.A }
			`),
		),
	)
}

func TestGoModTidyAddsGoDirectiveToMinimalModule(t *testing.T) {
	// given a bare go.mod with only a module directive.
	spec := test.NewRecipeSpec().WithRecipe(&migration.GoModTidy{})
	resolved := []golang.GoResolvedDependency{
		{ModulePath: "example.com/app", Main: true},
	}

	// when / then a `go` directive should be added, as `go mod tidy` does.
	spec.RewriteRun(t,
		test.GoProject("app",
			resolvedGraph(
				test.GoMod(
					`
					module example.com/app
				`,
					fmt.Sprintf(`
					module example.com/app

					go %s
				`, migration.DefaultGoDirectiveVersion())),
				resolved, nil,
			),
			test.Golang(`
				package main

				func main() {}
			`),
		),
	)
}

func TestGoModTidyNoEditWhenResolutionIncomplete(t *testing.T) {
	// given resolution ran (the build list is present) but the package->module map
	// was withheld because a module could not be fetched, so PackageModules is empty;
	// go-shared-libraries is imported (used) and unused/mod is an unreachable removal
	// candidate that a complete map would drop.
	spec := test.NewRecipeSpec().WithRecipe(&migration.GoModTidy{})
	resolved := []golang.GoResolvedDependency{
		{ModulePath: "example.com/app", Main: true},
		{ModulePath: "github.com/cof-primary/go-shared-libraries", Version: "v1.2.3"},
		{ModulePath: "github.com/unused/mod", Version: "v1.0.0", Indirect: true},
	}

	// when / then no edits at all: with no trustworthy map, nothing is removed, added, or re-marked.
	spec.RewriteRun(t,
		test.GoProject("app",
			graphWithStatus(
				test.GoMod(`
					module example.com/app

					go 1.22

					require (
						github.com/cof-primary/go-shared-libraries v1.2.3
						github.com/unused/mod v1.0.0 // indirect
					)
				`),
				resolved, nil, golang.GoResolutionIncomplete,
			),
			test.Golang(`
				package main

				import "github.com/cof-primary/go-shared-libraries/gotel"

				func main() { _ = gotel.Name }
			`),
		),
	)
}

func TestGoModTidyNoEditWhenGoSumOnly(t *testing.T) {
	// given a complete-looking build list and package->module map, but resolution
	// fell back to go.sum (status GO_SUM_ONLY) so the graph must not be trusted:
	// baz/qux would be added under a trustworthy resolution.
	spec := test.NewRecipeSpec().WithRecipe(&migration.GoModTidy{})
	resolved := []golang.GoResolvedDependency{
		{ModulePath: "example.com/app", Main: true},
		{ModulePath: "github.com/foo/bar", Version: "v1.0.0"},
		{ModulePath: "github.com/baz/qux", Version: "v1.0.0", Indirect: true},
	}
	pkgs := []golang.GoPackageModule{
		{ImportPath: "github.com/foo/bar", ModulePath: "github.com/foo/bar", Version: "v1.0.0"},
		{ImportPath: "github.com/baz/qux", ModulePath: "github.com/baz/qux", Version: "v1.0.0"},
	}

	// when / then no edits: the graph-dependent steps are gated off GO_SUM_ONLY.
	spec.RewriteRun(t,
		test.GoProject("app",
			graphWithStatus(
				test.GoMod(`
					module example.com/app

					go 1.22

					require github.com/foo/bar v1.0.0
				`),
				resolved, pkgs, golang.GoResolutionGoSumOnly,
			),
			test.Golang(`
				package main

				import "github.com/foo/bar"

				func main() { _ = bar.A }
			`),
		),
	)
}
