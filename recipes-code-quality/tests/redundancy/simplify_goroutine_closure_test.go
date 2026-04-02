/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package redundancy_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/redundancy"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestSimplifyGoroutineClosureSingleCall(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.SimplifyGoroutineClosure{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func doWork() {}

			func f() {
				go func() { doWork() }()
			}
		`, `
			package main

			func doWork() {}

			func f() {
				go doWork()
			}
		`),
	)
}

func TestSimplifyGoroutineClosureNoChangeMultipleStatements(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.SimplifyGoroutineClosure{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func setup() {}
			func doWork() {}

			func f() {
				go func() {
					setup()
					doWork()
				}()
			}
		`),
	)
}

func TestSimplifyGoroutineClosureNoChangeDirectCall(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.SimplifyGoroutineClosure{})
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
