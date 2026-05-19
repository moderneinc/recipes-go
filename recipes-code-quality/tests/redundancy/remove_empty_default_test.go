/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package redundancy_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes-code-quality/recipes/redundancy"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestRemoveEmptyDefaultSimple(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.RemoveEmptyDefault{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(x int) {
				switch x {
				case 1:
					println("one")
				default:
				}
			}
		`, `
			package main

			func f(x int) {
				switch x {
				case 1:
					println("one")
				}
			}
		`),
	)
}

func TestRemoveEmptyDefaultNoChangeWithBody(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.RemoveEmptyDefault{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(x int) {
				switch x {
				case 1:
					println("one")
				default:
					return
				}
			}
		`),
	)
}

func TestRemoveEmptyDefaultNoChangeNonDefaultEmpty(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.RemoveEmptyDefault{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(x int) {
				switch x {
				case 1:
				default:
					println("default")
				}
			}
		`),
	)
}
