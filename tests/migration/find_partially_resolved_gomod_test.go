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

// TestGoModTidyNoEditButSignalWhenGoSumOnly is the I6 companion to
// TestGoModTidyNoEditWhenGoSumOnly: GoModTidy still makes no edit (its offline
// no-op is safe), but FindPartiallyResolvedGoMod surfaces the skipped tidy so the
// run is not silently indistinguishable from an already-tidy module.
func TestGoModTidyNoEditButSignalWhenGoSumOnly(t *testing.T) {
	// given a module whose go.mod imports a package it does not require, but whose
	// graph resolved from go.sum alone (GO_SUM_ONLY), so add/remove are skipped.
	spec := test.NewRecipeSpec().WithRecipe(&migration.FindPartiallyResolvedGoMod{})
	resolved := []golang.GoResolvedDependency{
		{ModulePath: "example.com/app", Main: true},
		{ModulePath: "github.com/foo/bar", Version: "v1.0.0"},
	}

	// when / then the module directive is flagged, naming why the tidy stopped short.
	spec.RewriteRun(t,
		test.GoProject("app",
			graphWithUnresolved(
				test.GoMod(`
					module example.com/app

					go 1.22

					require github.com/foo/bar v1.0.0
				`, `
					/*~~(go.mod not fully resolved offline (GO_SUM_ONLY); go mod tidy add/remove skipped)~~>*/module example.com/app

					go 1.22

					require github.com/foo/bar v1.0.0
				`),
				resolved, nil, golang.GoResolutionGoSumOnly, nil, "reading go.sum: missing go.sum entry for github.com/baz/qux",
			),
			test.Golang(`
				package main

				import "github.com/foo/bar"

				func main() { _ = bar.A }
			`),
		),
	)
}

// TestFindPartiallyResolvedGoModFlagsIncomplete covers the INCOMPLETE status,
// where the toolchain names the imports it could not map.
func TestFindPartiallyResolvedGoModFlagsIncomplete(t *testing.T) {
	// given a graph resolved but for a package the toolchain could not map to a module.
	spec := test.NewRecipeSpec().WithRecipe(&migration.FindPartiallyResolvedGoMod{})
	resolved := []golang.GoResolvedDependency{
		{ModulePath: "example.com/app", Main: true},
	}

	// when / then the module directive is flagged for INCOMPLETE resolution.
	spec.RewriteRun(t,
		test.GoProject("app",
			graphWithUnresolved(
				test.GoMod(`
					module example.com/app

					go 1.22
				`, `
					/*~~(go.mod not fully resolved offline (INCOMPLETE); go mod tidy add/remove skipped)~~>*/module example.com/app

					go 1.22
				`),
				resolved, nil, golang.GoResolutionIncomplete, []string{"github.com/redis/go-redis/v9"}, "",
			),
			test.Golang(`
				package main

				import "github.com/redis/go-redis/v9"

				func main() { _ = redis.Nil }
			`),
		),
	)
}

// TestFindPartiallyResolvedGoModIgnoresResolved confirms a fully resolved go.mod
// is not flagged: GoModTidy handles it, so there is nothing to surface.
func TestFindPartiallyResolvedGoModIgnoresResolved(t *testing.T) {
	// given a fully resolved graph with a package->module map
	spec := test.NewRecipeSpec().WithRecipe(&migration.FindPartiallyResolvedGoMod{})
	resolved := []golang.GoResolvedDependency{
		{ModulePath: "example.com/app", Main: true},
		{ModulePath: "github.com/foo/bar", Version: "v1.0.0"},
	}
	pkgs := []golang.GoPackageModule{
		{ImportPath: "github.com/foo/bar", ModulePath: "github.com/foo/bar", Version: "v1.0.0"},
	}

	// when / then nothing is flagged
	spec.RewriteRun(t,
		test.GoProject("app",
			resolvedGraph(
				test.GoMod(`
					module example.com/app

					go 1.22

					require github.com/foo/bar v1.0.0
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
