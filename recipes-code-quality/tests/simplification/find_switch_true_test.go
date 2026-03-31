/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/simplification"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestFindSwitchTrue(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.FindSwitchTrue{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(x int) {
				switch true {
				case x > 0:
					println("positive")
				case x < 0:
					println("negative")
				}
			}
		`, `
			package main

			func f(x int) {
				switch {
				case x > 0:
					println("positive")
				case x < 0:
					println("negative")
				}
			}
		`),
	)
}

func TestFindSwitchTrueNoChangeTagless(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.FindSwitchTrue{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(x int) {
				switch {
				case x > 0:
					println("positive")
				case x < 0:
					println("negative")
				}
			}
		`),
	)
}

func TestFindSwitchTrueNoChangeVariable(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.FindSwitchTrue{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(x int) {
				switch x {
				case 1:
					println("one")
				case 2:
					println("two")
				}
			}
		`),
	)
}
