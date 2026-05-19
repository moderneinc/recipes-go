/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package redundancy_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes-code-quality/recipes/redundancy"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestRemoveRedundantElseSimple(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.RemoveRedundantElse{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(x int) int {
				if x > 0 {
					return x
				} else {
					x = -x
				}
				return x
			}
		`, `
			package main

			func f(x int) int {
				if x > 0 {
					return x
				}
				x = -x
				return x
			}
		`),
	)
}

func TestRemoveRedundantElseMultipleStatements(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.RemoveRedundantElse{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(x int) string {
				if x > 0 {
					println("positive")
					return "pos"
				} else {
					println("non-positive")
				}
				return "done"
			}
		`, `
			package main

			func f(x int) string {
				if x > 0 {
					println("positive")
					return "pos"
				}
				println("non-positive")
				return "done"
			}
		`),
	)
}

func TestRemoveRedundantElseNoChangeNoElse(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.RemoveRedundantElse{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(x int) int {
				if x > 0 {
					return x
				}
				return -x
			}
		`),
	)
}

func TestRemoveRedundantElseNoChangeNoReturn(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.RemoveRedundantElse{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(x int) int {
				if x > 0 {
					x = x + 1
				} else {
					x = x - 1
				}
				return x
			}
		`),
	)
}
