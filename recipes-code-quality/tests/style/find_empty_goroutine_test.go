/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/style"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestFindEmptyGoroutine(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindEmptyGoroutine{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() {
				go func() {}()
			}
		`),
	)
}

func TestFindEmptyGoroutineNoChangeWithBody(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindEmptyGoroutine{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() {
				go func() {
					doWork()
				}()
			}
		`),
	)
}
