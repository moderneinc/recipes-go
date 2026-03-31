/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/errorhandling"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestFindErrorTypeAssertionFound(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.FindErrorTypeAssertion{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			type MyError struct{ msg string }

			func (e *MyError) Error() string { return e.msg }

			func f(err error) {
				_ = err.(*MyError)
			}
		`),
	)
}

func TestFindErrorTypeAssertionNoChangeNonErr(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.FindErrorTypeAssertion{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(x interface{}) {
				_ = x.(int)
			}
		`),
	)
}
