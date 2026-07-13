/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package migration_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes/migration"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestFormatGoModSortsRequireBlock(t *testing.T) {
	// given an out-of-order require block
	spec := test.NewRecipeSpec().WithRecipe(&migration.FormatGoMod{})

	// when / then entries sort by module path, // indirect travels with its entry
	spec.RewriteRun(t,
		test.GoMod(`
			module example.com/app

			go 1.22

			require (
				github.com/foo/bar v1.2.3
				github.com/baz/qux v1.0.0 // indirect
				example.com/aaa v0.1.0
			)
		`, `
			module example.com/app

			go 1.22

			require (
				example.com/aaa v0.1.0
				github.com/baz/qux v1.0.0 // indirect
				github.com/foo/bar v1.2.3
			)
		`),
	)
}

func TestFormatGoModNoChangeWhenSorted(t *testing.T) {
	// given an already-sorted require block
	spec := test.NewRecipeSpec().WithRecipe(&migration.FormatGoMod{})

	// when / then no change
	spec.RewriteRun(t,
		test.GoMod(`
			module example.com/app

			go 1.22

			require (
				example.com/aaa v0.1.0
				github.com/baz/qux v1.0.0 // indirect
				github.com/foo/bar v1.2.3
			)
		`),
	)
}

func TestFormatGoModSkipsBlockWithLineComments(t *testing.T) {
	// given a require block with a standalone comment line above an entry
	spec := test.NewRecipeSpec().WithRecipe(&migration.FormatGoMod{})

	// when / then the block is left untouched to avoid detaching the comment
	spec.RewriteRun(t,
		test.GoMod(`
			module example.com/app

			go 1.22

			require (
				github.com/foo/bar v1.2.3
				// keep this pinned
				example.com/aaa v0.1.0
			)
		`),
	)
}
