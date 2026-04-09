/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package redundancy_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/redundancy"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestRemoveDuplicateConditionsSimple(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.RemoveDuplicateConditions{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(x int) {
				if x > 0 {
					println("a")
				} else if x > 0 {
					println("b")
				}
			}
		`, `
			package main

			func f(x int) {
				if x > 0 {
					println("a")
				}
			}
		`),
	)
}

func TestRemoveDuplicateConditionsWithElse(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.RemoveDuplicateConditions{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(x int) {
				if x > 0 {
					println("a")
				} else if x > 0 {
					println("b")
				} else {
					println("c")
				}
			}
		`, `
			package main

			func f(x int) {
				if x > 0 {
					println("a")
				} else {
					println("c")
				}
			}
		`),
	)
}

func TestRemoveDuplicateConditionsMiddleBranch(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.RemoveDuplicateConditions{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(x int) {
				if x > 0 {
					println("a")
				} else if x < 0 {
					println("b")
				} else if x > 0 {
					println("c")
				} else {
					println("d")
				}
			}
		`, `
			package main

			func f(x int) {
				if x > 0 {
					println("a")
				} else if x < 0 {
					println("b")
				} else {
					println("d")
				}
			}
		`),
	)
}

func TestRemoveDuplicateConditionsNoChange(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.RemoveDuplicateConditions{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(x int) {
				if x > 0 {
					println("a")
				} else if x < 0 {
					println("b")
				} else {
					println("c")
				}
			}
		`),
	)
}

func TestRemoveDuplicateConditionsNoChangeNoElseIf(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.RemoveDuplicateConditions{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(x int) {
				if x > 0 {
					println("a")
				} else {
					println("b")
				}
			}
		`),
	)
}
