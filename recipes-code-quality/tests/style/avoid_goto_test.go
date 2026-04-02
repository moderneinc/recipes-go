/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/style"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestAvoidGoto(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.AvoidGoto{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() {
				goto done
			done:
				println("done")
			}
		`),
	)
}

func TestAvoidGotoNoChangeWithoutGoto(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.AvoidGoto{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() {
				println("hello")
			}
		`),
	)
}
