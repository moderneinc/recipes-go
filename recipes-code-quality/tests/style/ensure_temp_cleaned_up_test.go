/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/style"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestEnsureTempCleanedUpCreateTemp(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.EnsureTempCleanedUp{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "os"

			func f() {
				f, err := os.CreateTemp("", "prefix")
				_ = err
				_ = f
			}
		`, `
			package main

			import "os"

			func f() {
				f, err := os.CreateTemp("", "prefix")
				defer os.Remove(f.Name())
				_ = err
				_ = f
			}
		`),
	)
}

func TestEnsureTempCleanedUpNoChangeOpen(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.EnsureTempCleanedUp{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "os"

			func f() {
				os.Open("file")
			}
		`),
	)
}

func TestEnsureTempCleanedUpAlreadyDeferred(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.EnsureTempCleanedUp{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "os"

			func f() {
				f, err := os.CreateTemp("", "prefix")
				defer os.Remove(f.Name())
				_ = err
			}
		`),
	)
}
