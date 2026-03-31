/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package redundancy_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/redundancy"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestFindRedundantGoroutineClosureSingleCall(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.FindRedundantGoroutineClosure{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func doWork() {}

			func f() {
				go func() { doWork() }()
			}
		`),
	)
}

func TestFindRedundantGoroutineClosureNoChangeMultipleStatements(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.FindRedundantGoroutineClosure{})
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

func TestFindRedundantGoroutineClosureNoChangeDirectCall(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.FindRedundantGoroutineClosure{})
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
