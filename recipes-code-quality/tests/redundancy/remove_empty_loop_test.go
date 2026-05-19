/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package redundancy_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes-code-quality/recipes/redundancy"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestRemoveEmptyLoopBareFor(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.RemoveEmptyLoop{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() {
				for {
				}
			}
		`, `
			package main

			func f() {
			}
		`),
	)
}

func TestRemoveEmptyLoopThreeClause(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.RemoveEmptyLoop{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(n int) {
				for i := 0; i < n; i++ {
				}
			}
		`, `
			package main

			func f(n int) {
			}
		`),
	)
}

func TestRemoveEmptyLoopNoChangeWithBody(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.RemoveEmptyLoop{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func doWork() {}

			func f() {
				for {
					doWork()
				}
			}
		`),
	)
}
