/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes-code-quality/recipes/style"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestAvoidNestedGoroutine(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.AvoidNestedGoroutine{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() {
				go func() {
					go doMore()
				}()
			}
		`, `
			package main

			func f() {
				go func() {
					/*~~(nested goroutine; consider restructuring to avoid goroutines inside goroutines)~~>*/go doMore()
				}()
			}
		`),
	)
}

func TestAvoidNestedGoroutineNoChangeSingle(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.AvoidNestedGoroutine{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() {
				go doWork()
			}
		`),
	)
}
