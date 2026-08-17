/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package testify_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes/migration/testify"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestAddTestifyDependencyToExistingBlock(t *testing.T) {
	// given a module that imports testify but does not require it
	spec := test.NewRecipeSpec().WithRecipe(&testify.AddTestifyDependency{})

	// when / then the require is appended to the existing block
	spec.RewriteRun(t,
		test.GoProject("app",
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
					github.com/stretchr/testify v1.10.0
				)
			`),
			test.Golang(`
				package app

				import (
					"testing"

					"github.com/stretchr/testify/require"
				)

				func TestThing(t *testing.T) {
					require.NoError(t, do())
				}

				func do() error { return nil }
			`),
		),
	)
}

// No testify import anywhere: go.mod is untouched.
func TestAddTestifyDependencyNoChangeWhenNotImported(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&testify.AddTestifyDependency{})
	spec.RewriteRun(t,
		test.GoProject("app",
			test.GoMod(`
				module example.com/app

				go 1.22

				require github.com/foo/bar v1.2.3
			`),
			test.Golang(`
				package app

				func do() error { return nil }
			`),
		),
	)
}

// testify already required: no duplicate is added.
func TestAddTestifyDependencyNoChangeWhenAlreadyRequired(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&testify.AddTestifyDependency{})
	spec.RewriteRun(t,
		test.GoProject("app",
			test.GoMod(`
				module example.com/app

				go 1.22

				require github.com/stretchr/testify v1.10.0
			`),
			test.Golang(`
				package app

				import "github.com/stretchr/testify/require"

				var _ = require.NoError
			`),
		),
	)
}
