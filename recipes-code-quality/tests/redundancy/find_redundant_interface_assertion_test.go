/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package redundancy_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/redundancy"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestFindRedundantInterfaceAssertionAny(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.FindRedundantInterfaceAssertion{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(x interface{}) {
				_ = x.(any)
			}
		`),
	)
}

func TestFindRedundantInterfaceAssertionNoChangeInt(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.FindRedundantInterfaceAssertion{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(x interface{}) {
				_ = x.(int)
			}
		`),
	)
}

func TestFindRedundantInterfaceAssertionNoChangeString(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.FindRedundantInterfaceAssertion{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(x interface{}) {
				_ = x.(string)
			}
		`),
	)
}
