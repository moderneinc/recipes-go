/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/simplification"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestSimplifyBooleanExpressionEqualsTrue(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.SimplifyBooleanExpression{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(ok bool) {
				if ok == true {
					println("yes")
				}
			}
		`, `
			package main

			func f(ok bool) {
				if ok {
					println("yes")
				}
			}
		`),
	)
}

func TestSimplifyBooleanExpressionEqualsFalse(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.SimplifyBooleanExpression{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(ok bool) {
				if ok == false {
					println("no")
				}
			}
		`, `
			package main

			func f(ok bool) {
				if !ok {
					println("no")
				}
			}
		`),
	)
}

func TestSimplifyBooleanExpressionTrueEquals(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.SimplifyBooleanExpression{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(ok bool) {
				if true == ok {
					println("yes")
				}
			}
		`, `
			package main

			func f(ok bool) {
				if ok {
					println("yes")
				}
			}
		`),
	)
}

func TestSimplifyBooleanExpressionNotEqualsTrue(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.SimplifyBooleanExpression{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(ok bool) {
				if ok != true {
					println("no")
				}
			}
		`, `
			package main

			func f(ok bool) {
				if !ok {
					println("no")
				}
			}
		`),
	)
}

func TestSimplifyBooleanExpressionNoChangeComplex(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.SimplifyBooleanExpression{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(a, b bool) {
				if a && b {
					println("both")
				}
			}
		`),
	)
}
