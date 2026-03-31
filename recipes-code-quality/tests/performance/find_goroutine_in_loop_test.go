/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package performance_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/performance"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestFindGoroutineInForLoop(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&performance.FindGoroutineInLoop{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func doWork() {}

			func f() {
				for i := 0; i < 10; i++ {
					go doWork()
				}
			}
		`),
	)
}

func TestFindGoroutineInRangeLoop(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&performance.FindGoroutineInLoop{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func process(s string) {}

			func f(items []string) {
				for _, item := range items {
					go process(item)
				}
			}
		`),
	)
}

func TestFindGoroutineNoChangeOutsideLoop(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&performance.FindGoroutineInLoop{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func doWork() {}

			func f() {
				go doWork()
			}
		`),
	)
}
