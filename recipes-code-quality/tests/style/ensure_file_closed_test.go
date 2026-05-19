/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes-code-quality/recipes/style"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestEnsureFileClosed(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.EnsureFileClosed{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "os"

			func f() {
				f, err := os.Open("file")
				_ = err
				_ = f
			}
		`, `
			package main

			import "os"

			func f() {
				f, err := os.Open("file")
				defer f.Close()
				_ = err
				_ = f
			}
		`),
	)
}

func TestEnsureFileClosedCreate(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.EnsureFileClosed{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "os"

			func f() {
				f, err := os.Create("file")
				_ = err
				_ = f
			}
		`, `
			package main

			import "os"

			func f() {
				f, err := os.Create("file")
				defer f.Close()
				_ = err
				_ = f
			}
		`),
	)
}

func TestEnsureFileClosedNoChange(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.EnsureFileClosed{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "os"

			func f() {
				os.Getenv("X")
			}
		`),
	)
}

func TestEnsureFileClosedAlreadyDeferred(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.EnsureFileClosed{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "os"

			func f() {
				f, err := os.Open("file")
				defer f.Close()
				_ = err
			}
		`),
	)
}
