/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package migration_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes/migration"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestFindUnusedGoModRequiresFlagsDirectUnimported(t *testing.T) {
	// given a direct require that nothing imports, alongside an imported one
	spec := test.NewRecipeSpec().WithRecipe(&migration.FindUnusedGoModRequires{})

	// when / then only the unimported direct require is flagged
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
					/*~~(unused go.mod requirement)~~>*/github.com/baz/qux v1.0.0
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

func TestFindUnusedGoModRequiresIgnoresIndirect(t *testing.T) {
	// given an unimported require that is already marked // indirect
	spec := test.NewRecipeSpec().WithRecipe(&migration.FindUnusedGoModRequires{})

	// when / then nothing is flagged (indirect requires are not reported)
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

func TestFindUnusedGoModRequiresSingleLine(t *testing.T) {
	// given a single-line direct require that nothing imports
	spec := test.NewRecipeSpec().WithRecipe(&migration.FindUnusedGoModRequires{})

	// when / then the require is flagged
	spec.RewriteRun(t,
		test.GoProject("app",
			test.GoMod(`
				module example.com/app

				go 1.22

				require github.com/baz/qux v1.0.0
			`, `
				module example.com/app

				go 1.22

				/*~~(unused go.mod requirement)~~>*/require github.com/baz/qux v1.0.0
			`),
			test.Golang(`
				package main

				func main() {}
			`),
		),
	)
}
