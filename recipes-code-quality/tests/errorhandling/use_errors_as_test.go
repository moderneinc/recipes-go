/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/errorhandling"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestUseErrorsAsCommaOkIf(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.UseErrorsAs{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			type MyError struct{ msg string }

			func (e *MyError) Error() string { return e.msg }

			func f(err error) {
				if myErr, ok := err.(*MyError); ok {
					println(myErr.msg)
				}
			}
		`, `
			package main

			type MyError struct{ msg string }

			func (e *MyError) Error() string { return e.msg }

			func f(err error) {
				var myErr *MyError
				if errors.As(err, &myErr) {
					println(myErr.msg)
				}
			}
		`),
	)
}

func TestUseErrorsAsNoChangeNonCommaOk(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.UseErrorsAs{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(x interface{}) {
				if v, ok := x.(int); ok {
					println(v)
				}
			}
		`),
	)
}

func TestUseErrorsAsNoChangeNoInit(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.UseErrorsAs{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(cond bool) {
				if cond {
					println("yes")
				}
			}
		`),
	)
}
