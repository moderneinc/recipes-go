/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/style"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestFindTypeAssertionWithoutOk(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindTypeAssertionWithoutOk{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(x interface{}) {
				_ = x.(int)
			}
		`),
	)
}

func TestFindTypeAssertionWithoutOkNoChangeNoAssertion(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindTypeAssertionWithoutOk{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(x int) {
				_ = x + 1
			}
		`),
	)
}
