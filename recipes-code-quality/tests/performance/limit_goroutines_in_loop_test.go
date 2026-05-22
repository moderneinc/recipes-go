/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package performance_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes-code-quality/recipes/performance"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestFindGoroutineInForLoop(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&performance.LimitGoroutinesInLoop{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func doWork() {}

			func f() {
				for i := 0; i < 10; i++ {
					go doWork()
				}
			}
		`, `
			package main

			func doWork() {}

			func f() {
				for i := 0; i < 10; i++ {
					/*~~(goroutine launched in loop; unbounded goroutine creation can cause resource exhaustion)~~>*/go doWork()
				}
			}
		`),
	)
}

func TestFindGoroutineInRangeLoop(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&performance.LimitGoroutinesInLoop{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func process(s string) {}

			func f(items []string) {
				for _, item := range items {
					go process(item)
				}
			}
		`, `
			package main

			func process(s string) {}

			func f(items []string) {
				for _, item := range items {
					/*~~(goroutine launched in loop; unbounded goroutine creation can cause resource exhaustion)~~>*/go process(item)
				}
			}
		`),
	)
}

func TestFindGoroutineNoChangeOutsideLoop(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&performance.LimitGoroutinesInLoop{})
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
