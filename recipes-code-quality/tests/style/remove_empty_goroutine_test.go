/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes-code-quality/recipes/style"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestRemoveEmptyGoroutine(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.RemoveEmptyGoroutine{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() {
				go func() {}()
			}
		`, `
			package main

			func f() {
			}
		`),
	)
}

func TestRemoveEmptyGoroutineNoChangeWithBody(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.RemoveEmptyGoroutine{})
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
