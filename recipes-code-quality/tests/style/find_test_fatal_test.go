/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/style"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestFindTestFatal(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindTestFatal{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "testing"

			func TestFoo(t *testing.T) {
				t.Fatal("fail")
			}
		`),
	)
}

func TestFindTestFatalf(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindTestFatal{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "testing"

			func TestFoo(t *testing.T) {
				t.Fatalf("got %d", 1)
			}
		`),
	)
}

func TestFindTestFatalNoChangeError(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindTestFatal{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "testing"

			func TestFoo(t *testing.T) {
				t.Error("fail")
			}
		`),
	)
}
