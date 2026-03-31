/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package redundancy_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/redundancy"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestFindRedundantElseSimple(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.FindRedundantElse{})
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
		`),
	)
}

func TestFindRedundantElseMultipleStatements(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.FindRedundantElse{})
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
		`),
	)
}

func TestFindRedundantElseNoChangeNoElse(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.FindRedundantElse{})
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

func TestFindRedundantElseNoChangeNoReturn(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.FindRedundantElse{})
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
