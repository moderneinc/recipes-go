/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package redundancy_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/redundancy"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestFindEmptySwitchTagless(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.FindEmptySwitch{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() {
				switch {
				}
			}
		`),
	)
}

func TestFindEmptySwitchWithTag(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.FindEmptySwitch{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(x int) {
				switch x {
				}
			}
		`),
	)
}

func TestFindEmptySwitchNoChangeWithCase(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.FindEmptySwitch{})
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

func TestFindEmptySwitchNoChangeWithDefault(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.FindEmptySwitch{})
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
