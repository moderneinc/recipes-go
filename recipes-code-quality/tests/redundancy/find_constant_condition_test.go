/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package redundancy_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/redundancy"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestFindConstantConditionTrue(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.FindConstantCondition{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() {
				if true {
					println("always")
				}
			}
		`),
	)
}

func TestFindConstantConditionFalse(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.FindConstantCondition{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() {
				if false {
					println("never")
				}
			}
		`),
	)
}

func TestFindConstantConditionNoChangeVariable(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.FindConstantCondition{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(ok bool) {
				if ok {
					println("maybe")
				}
			}
		`),
	)
}
