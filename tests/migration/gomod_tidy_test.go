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

	// when / then: baz/qux added (// indirect), dead/mod removed, block sorted.
	spec.RewriteRun(t,
		test.GoProject("app",
			test.GoModGraph(
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
						github.com/baz/qux v1.0.0 // indirect
						github.com/foo/bar v1.0.0
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
