/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package migration_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes/migration"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestFixGoModIndirectStripsMarkerFromImportedModule(t *testing.T) {
	// given a require marked // indirect that the module actually imports
	spec := test.NewRecipeSpec().WithRecipe(&migration.FixGoModIndirectMarkers{})

	// when / then the imported one becomes direct, the unused one stays indirect
	spec.RewriteRun(t,
		test.GoProject("app",
			test.GoMod(`
				module example.com/app

				go 1.22

				require (
					github.com/foo/bar v1.2.3 // indirect
					github.com/baz/qux v1.0.0 // indirect
				)
			`, `
				module example.com/app

				go 1.22

				require (
					github.com/foo/bar v1.2.3
					github.com/baz/qux v1.0.0 // indirect
				)
			`),
			test.Golang(`
				package main

				import "github.com/foo/bar"

				func main() { _ = bar.A }
			`),
		),
	)
}

func TestFixGoModIndirectAddsMarkerToUnusedModule(t *testing.T) {
	// given a direct require that nothing imports, and a sub-package import
	spec := test.NewRecipeSpec().WithRecipe(&migration.FixGoModIndirectMarkers{})

	// when / then the unimported module gains // indirect; the sub-package import keeps its module direct
	spec.RewriteRun(t,
		test.GoProject("app",
			test.GoMod(`
				module example.com/app

				go 1.22

				require (
					github.com/foo/bar v1.2.3
					github.com/baz/qux v1.0.0
				)
			`, `
				module example.com/app

				go 1.22

				require (
					github.com/foo/bar v1.2.3
					github.com/baz/qux v1.0.0 // indirect
				)
			`),
			test.Golang(`
				package main

				import "github.com/foo/bar/sub"

				func main() { _ = sub.A }
			`),
		),
	)
}

func TestFixGoModIndirectSingleLineRequire(t *testing.T) {
	// given a single-line require that is imported
	spec := test.NewRecipeSpec().WithRecipe(&migration.FixGoModIndirectMarkers{})

	// when / then the // indirect marker is stripped
	spec.RewriteRun(t,
		test.GoProject("app",
			test.GoMod(`
				module example.com/app

				go 1.22

				require github.com/foo/bar v1.2.3 // indirect
			`, `
				module example.com/app

				go 1.22

				require github.com/foo/bar v1.2.3
			`),
			test.Golang(`
				package main

				import "github.com/foo/bar"

				func main() { _ = bar.A }
			`),
		),
	)
}

func TestFixGoModIndirectNestedModulesBindToLongestPath(t *testing.T) {
	// given a parent and a nested child module, importing only the child
	spec := test.NewRecipeSpec().WithRecipe(&migration.FixGoModIndirectMarkers{})

	// when / then the import binds to the nested module only; the parent becomes indirect
	spec.RewriteRun(t,
		test.GoProject("app",
			test.GoMod(`
				module example.com/app

				go 1.22

				require (
					github.com/foo/bar v1.2.3
					github.com/foo/bar/nested v1.0.0
				)
			`, `
				module example.com/app

				go 1.22

				require (
					github.com/foo/bar v1.2.3 // indirect
					github.com/foo/bar/nested v1.0.0
				)
			`),
			test.Golang(`
				package main

				import "github.com/foo/bar/nested/pkg"

				func main() { _ = pkg.A }
			`),
		),
	)
}

func TestFixGoModIndirectNoChangeWhenAlreadyCorrect(t *testing.T) {
	// given a go.mod whose markers already match usage
	spec := test.NewRecipeSpec().WithRecipe(&migration.FixGoModIndirectMarkers{})

	// when / then no change
	spec.RewriteRun(t,
		test.GoProject("app",
			test.GoMod(`
				module example.com/app

				go 1.22

				require (
					github.com/foo/bar v1.2.3
					github.com/baz/qux v1.0.0 // indirect
				)
			`),
			test.Golang(`
				package main

				import "github.com/foo/bar"

				func main() { _ = bar.A }
			`),
		),
	)
}
