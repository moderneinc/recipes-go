/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package redundancy_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/redundancy"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestFindEmptyDefaultSimple(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.FindEmptyDefault{})
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
		`),
	)
}

func TestFindEmptyDefaultNoChangeWithBody(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.FindEmptyDefault{})
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

func TestFindEmptyDefaultNoChangeNonDefaultEmpty(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.FindEmptyDefault{})
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
