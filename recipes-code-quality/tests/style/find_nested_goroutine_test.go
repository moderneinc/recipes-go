/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/style"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestFindNestedGoroutine(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindNestedGoroutine{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() {
				go func() {
					go doMore()
				}()
			}
		`),
	)
}

func TestFindNestedGoroutineNoChangeSingle(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindNestedGoroutine{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() {
				go doWork()
			}
		`),
	)
}
