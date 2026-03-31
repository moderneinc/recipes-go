/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package redundancy_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/redundancy"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestFindEmptyLoopBareFor(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.FindEmptyLoop{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() {
				for {
				}
			}
		`),
	)
}

func TestFindEmptyLoopThreeClause(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.FindEmptyLoop{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(n int) {
				for i := 0; i < n; i++ {
				}
			}
		`),
	)
}

func TestFindEmptyLoopNoChangeWithBody(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.FindEmptyLoop{})
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
