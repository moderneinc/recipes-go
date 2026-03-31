/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/errorhandling"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestFindEmptyErrorCheckSimple(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.FindEmptyErrorCheck{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() {
				err := doSomething()
				if err != nil {
				}
			}

			func doSomething() error { return nil }
		`),
	)
}

func TestFindEmptyErrorCheckNoChangeHandled(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.FindEmptyErrorCheck{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "log"

			func f() {
				err := doSomething()
				if err != nil {
					log.Fatal(err)
				}
			}

			func doSomething() error { return nil }
		`),
	)
}

func TestFindEmptyErrorCheckNoChangeNilCheck(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.FindEmptyErrorCheck{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() {
				x := 1
				if x != 0 {
				}
			}
		`),
	)
}
