/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package redundancy_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/redundancy"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestRemoveConstantConditionTrue(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.RemoveConstantCondition{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() {
				if true {
					println("always")
				}
			}
		`, `
			package main

			func f() {
				{
					println("always")
				}
			}
		`),
	)
}

func TestRemoveConstantConditionFalse(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.RemoveConstantCondition{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() {
				if false {
					println("never")
				}
			}
		`, `
			package main

			func f() {
			}
		`),
	)
}

func TestRemoveConstantConditionFalseWithElse(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.RemoveConstantCondition{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() {
				if false {
					println("never")
				} else {
					println("always")
				}
			}
		`, `
			package main

			func f() {
				{
					println("always")
				}
			}
		`),
	)
}

func TestRemoveConstantConditionNoChangeVariable(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.RemoveConstantCondition{})
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
