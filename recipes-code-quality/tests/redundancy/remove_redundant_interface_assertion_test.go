/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package redundancy_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes-code-quality/recipes/redundancy"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestRemoveRedundantInterfaceAssertionAny(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.RemoveRedundantInterfaceAssertion{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(x interface{}) {
				_ = x.(any)
			}
		`, `
			package main

			func f(x interface{}) {
				_ = x
			}
		`),
	)
}

func TestRemoveRedundantInterfaceAssertionNoChangeInt(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.RemoveRedundantInterfaceAssertion{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(x interface{}) {
				_ = x.(int)
			}
		`),
	)
}

func TestRemoveRedundantInterfaceAssertionNoChangeString(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.RemoveRedundantInterfaceAssertion{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(x interface{}) {
				_ = x.(string)
			}
		`),
	)
}
