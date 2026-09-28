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
	pkgs := []golang.GoPackageModule{
		{ImportPath: "github.com/foo/bar", ModulePath: "github.com/foo/bar", Version: "v1.2.3"},
		{ImportPath: "github.com/baz/qux", ModulePath: "github.com/baz/qux", Version: "v1.0.0"},
		{ImportPath: "golang.org/x/text/language", ModulePath: "golang.org/x/text", Version: "v0.3.0"},
	}

	// when / then the direct module joins the direct block and the indirect one
	//           lands in its own indirect block
	spec.RewriteRun(t,
		resolvedGraph(
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
				)

				require (
					golang.org/x/text v0.3.0 // indirect
				)
			`),
			resolved, pkgs,
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
	pkgs := []golang.GoPackageModule{
		{ImportPath: "github.com/foo/bar", ModulePath: "github.com/foo/bar", Version: "v1.2.3"},
		{ImportPath: "github.com/baz/qux", ModulePath: "github.com/baz/qux", Version: "v1.0.0"},
	}

	// when / then a new require block is created for the missing module
	spec.RewriteRun(t,
		resolvedGraph(
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
			resolved, pkgs,
		),
	)
}

func TestAddMissingGoModRequiresAddsIndirectToIndirectBlock(t *testing.T) {
	// given a go.mod with a direct block then an indirect block, and a resolved
	//       graph needing one new indirect module
	spec := test.NewRecipeSpec().WithRecipe(&migration.AddMissingGoModRequires{})
	resolved := []golang.GoResolvedDependency{
		{ModulePath: "example.com/app", Main: true},
		{ModulePath: "github.com/foo/bar", Version: "v1.2.3"},
		{ModulePath: "golang.org/x/sys", Version: "v0.1.0", Indirect: true},
		{ModulePath: "golang.org/x/text", Version: "v0.3.0", Indirect: true},
	}
	pkgs := []golang.GoPackageModule{
		{ImportPath: "github.com/foo/bar", ModulePath: "github.com/foo/bar", Version: "v1.2.3"},
		{ImportPath: "golang.org/x/sys/unix", ModulePath: "golang.org/x/sys", Version: "v0.1.0"},
		{ImportPath: "golang.org/x/text/language", ModulePath: "golang.org/x/text", Version: "v0.3.0"},
	}

	// when / then the new module is added to the indirect block, not the direct block
	spec.RewriteRun(t,
		resolvedGraph(
			test.GoMod(`
				module example.com/app

				go 1.22

				require (
					github.com/foo/bar v1.2.3
				)

				require (
					golang.org/x/sys v0.1.0 // indirect
				)
			`, `
				module example.com/app

				go 1.22

				require (
					github.com/foo/bar v1.2.3
				)

				require (
					golang.org/x/sys v0.1.0 // indirect
					golang.org/x/text v0.3.0 // indirect
				)
			`),
			resolved, pkgs,
		),
	)
}

func TestAddMissingGoModRequiresSkipsUnimportedPrunedModules(t *testing.T) {
	// given testify is imported but its (unimported) objx dep is in the build list
	spec := test.NewRecipeSpec().WithRecipe(&migration.AddMissingGoModRequires{})
	resolved := []golang.GoResolvedDependency{
		{ModulePath: "example.com/app", Main: true},
		{ModulePath: "github.com/stretchr/testify", Version: "v1.11.1", Deps: []golang.GoModuleRef{
			{ModulePath: "github.com/stretchr/objx", Version: "v0.5.2"},
		}},
		{ModulePath: "github.com/stretchr/objx", Version: "v0.5.2", Indirect: true},
	}
	pkgs := []golang.GoPackageModule{
		{ImportPath: "github.com/stretchr/testify/require", ModulePath: "github.com/stretchr/testify", Version: "v1.11.1"},
	}

	// when / then no require is added: objx is reachable but not imported
	spec.RewriteRun(t,
		resolvedGraph(
			test.GoMod(`
				module example.com/app

				go 1.25

				require github.com/stretchr/testify v1.11.1
			`),
			resolved, pkgs,
		),
	)
}

func TestAddMissingGoModRequiresSkipsImpliedIndirectBelowGo117(t *testing.T) {
	// given a go 1.15 main module that directly requires go-md2man, whose go.mod
	//       already requires blackfriday at the selected version, and blackfriday
	//       is a transitively imported indirect in the build list
	spec := test.NewRecipeSpec().WithRecipe(&migration.AddMissingGoModRequires{})
	resolved := []golang.GoResolvedDependency{
		{ModulePath: "example.com/app", Main: true},
		{ModulePath: "github.com/cpuguy83/go-md2man/v2", Version: "v2.0.6", Deps: []golang.GoModuleRef{
			{ModulePath: "github.com/russross/blackfriday/v2", Version: "v2.1.0"},
		}},
		{ModulePath: "github.com/russross/blackfriday/v2", Version: "v2.1.0", Indirect: true},
	}
	pkgs := []golang.GoPackageModule{
		{ImportPath: "github.com/cpuguy83/go-md2man/v2/md2man", ModulePath: "github.com/cpuguy83/go-md2man/v2", Version: "v2.0.6"},
		{ImportPath: "github.com/russross/blackfriday/v2", ModulePath: "github.com/russross/blackfriday/v2", Version: "v2.1.0"},
	}

	// when / then blackfriday is not added: go mod tidy below 1.17 omits an
	//           indirect already implied by a dependency's go.mod
	spec.RewriteRun(t,
		resolvedGraph(
			test.GoMod(`
				module example.com/app

				go 1.15

				require github.com/cpuguy83/go-md2man/v2 v2.0.6
			`),
			resolved, pkgs,
		),
	)
}

func TestAddMissingGoModRequiresAddsImpliedIndirectFromGo117(t *testing.T) {
	// given the same graph but a go 1.17 main module
	spec := test.NewRecipeSpec().WithRecipe(&migration.AddMissingGoModRequires{})
	resolved := []golang.GoResolvedDependency{
		{ModulePath: "example.com/app", Main: true},
		{ModulePath: "github.com/cpuguy83/go-md2man/v2", Version: "v2.0.6", Deps: []golang.GoModuleRef{
			{ModulePath: "github.com/russross/blackfriday/v2", Version: "v2.1.0"},
		}},
		{ModulePath: "github.com/russross/blackfriday/v2", Version: "v2.1.0", Indirect: true},
	}
	pkgs := []golang.GoPackageModule{
		{ImportPath: "github.com/cpuguy83/go-md2man/v2/md2man", ModulePath: "github.com/cpuguy83/go-md2man/v2", Version: "v2.0.6"},
		{ImportPath: "github.com/russross/blackfriday/v2", ModulePath: "github.com/russross/blackfriday/v2", Version: "v2.1.0"},
	}

	// when / then blackfriday is recorded: from go 1.17 tidy lists every
	//           transitively imported module explicitly
	spec.RewriteRun(t,
		resolvedGraph(
			test.GoMod(`
				module example.com/app

				go 1.17

				require github.com/cpuguy83/go-md2man/v2 v2.0.6
			`, `
				module example.com/app

				go 1.17

				require github.com/cpuguy83/go-md2man/v2 v2.0.6

				require (
					github.com/russross/blackfriday/v2 v2.1.0 // indirect
				)
			`),
			resolved, pkgs,
		),
	)
}

func TestAddMissingGoModRequiresAddsIndirectMainPinsBelowGo117(t *testing.T) {
	// given a go 1.15 main module where the imported indirect's selected version
	//       is higher than the version its requiring dependency declares, so the
	//       main module is what pins it
	spec := test.NewRecipeSpec().WithRecipe(&migration.AddMissingGoModRequires{})
	resolved := []golang.GoResolvedDependency{
		{ModulePath: "example.com/app", Main: true},
		{ModulePath: "github.com/foo/dep", Version: "v1.0.0", Deps: []golang.GoModuleRef{
			{ModulePath: "github.com/lone/x", Version: "v0.9.0"},
		}},
		{ModulePath: "github.com/lone/x", Version: "v1.0.0", Indirect: true},
	}
	pkgs := []golang.GoPackageModule{
		{ImportPath: "github.com/foo/dep", ModulePath: "github.com/foo/dep", Version: "v1.0.0"},
		{ImportPath: "github.com/lone/x", ModulePath: "github.com/lone/x", Version: "v1.0.0"},
	}

	// when / then lone/x is added: no dependency's go.mod implies v1.0.0
	spec.RewriteRun(t,
		resolvedGraph(
			test.GoMod(`
				module example.com/app

				go 1.15

				require github.com/foo/dep v1.0.0
			`, `
				module example.com/app

				go 1.15

				require github.com/foo/dep v1.0.0

				require (
					github.com/lone/x v1.0.0 // indirect
				)
			`),
			resolved, pkgs,
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
	pkgs := []golang.GoPackageModule{
		{ImportPath: "github.com/foo/bar", ModulePath: "github.com/foo/bar", Version: "v1.2.3"},
	}

	// when / then no change
	spec.RewriteRun(t,
		resolvedGraph(
			test.GoMod(`
				module example.com/app

				go 1.22

				require github.com/foo/bar v1.2.3
			`),
			resolved, pkgs,
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
