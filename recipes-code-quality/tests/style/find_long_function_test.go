/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/style"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestFindLongFunction(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindLongFunction{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() {
				x := 1
				x = x + 1
				x = x + 1
				x = x + 1
				x = x + 1
				x = x + 1
				x = x + 1
				x = x + 1
				x = x + 1
				x = x + 1
				x = x + 1
				x = x + 1
				x = x + 1
				x = x + 1
				x = x + 1
				x = x + 1
				x = x + 1
				x = x + 1
				x = x + 1
				x = x + 1
				x = x + 1
				x = x + 1
				x = x + 1
				x = x + 1
				_ = x
			}
		`),
	)
}

func TestFindLongFunctionNoChangeShort(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindLongFunction{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() {
				x := 1
				x = x + 1
				_ = x
			}
		`),
	)
}
