/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/style"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestAvoidFallthroughRemoved(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.AvoidFallthrough{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(x int) {
				switch x {
				case 1:
					println("one")
					fallthrough
				case 2:
					println("two")
				}
			}
		`, `
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

func TestAvoidFallthroughNoChangeNoFallthrough(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.AvoidFallthrough{})
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
