/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes-code-quality/recipes/style"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestKeepFunctionsShort(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.KeepFunctionsShort{})
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

func TestKeepFunctionsShortNoChangeShort(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.KeepFunctionsShort{})
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
