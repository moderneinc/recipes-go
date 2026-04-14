/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/style"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestUseCommaOkTypeAssertion(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.UseCommaOkTypeAssertion{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(x interface{}) {
				v := x.(int)
				_ = v
			}
		`, `
			package main

			func f(x interface{}) {
				v, ok := x.(int)
				_ = ok
				_ = v
			}
		`),
	)
}

func TestUseCommaOkTypeAssertionNoChangeNoAssertion(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.UseCommaOkTypeAssertion{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(x int) {
				_ = x + 1
			}
		`),
	)
}

func TestUseCommaOkTypeAssertionNoChangeBlankAssign(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.UseCommaOkTypeAssertion{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(x interface{}) {
				_ = x.(int)
			}
		`),
	)
}

func TestUseCommaOkTypeAssertionNoChangeAlreadyCommaOk(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.UseCommaOkTypeAssertion{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(x interface{}) {
				v, ok := x.(int)
				_, _ = v, ok
			}
		`),
	)
}
