/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package redundancy_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/redundancy"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestRemoveEmptySwitchTagless(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.RemoveEmptySwitch{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() {
				switch {
				}
			}
		`, `
			package main

			func f() {
			}
		`),
	)
}

func TestRemoveEmptySwitchWithTag(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.RemoveEmptySwitch{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(x int) {
				switch x {
				}
			}
		`, `
			package main

			func f(x int) {
			}
		`),
	)
}

func TestRemoveEmptySwitchNoChangeWithCase(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.RemoveEmptySwitch{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() {
				switch {
				case true:
					println("yes")
				}
			}
		`),
	)
}

func TestRemoveEmptySwitchNoChangeWithDefault(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.RemoveEmptySwitch{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(x int) {
				switch x {
				default:
					println("default")
				}
			}
		`),
	)
}
