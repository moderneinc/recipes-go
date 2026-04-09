/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/simplification"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestSimplifyRedundantLogicalAndIdent(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.SimplifyRedundantLogicalExpression{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(x bool) {
				if x && x {
					println("yes")
				}
			}
		`, `
			package main

			func f(x bool) {
				if x {
					println("yes")
				}
			}
		`),
	)
}

func TestSimplifyRedundantLogicalOrIdent(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.SimplifyRedundantLogicalExpression{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(x bool) {
				if x || x {
					println("yes")
				}
			}
		`, `
			package main

			func f(x bool) {
				if x {
					println("yes")
				}
			}
		`),
	)
}

func TestSimplifyRedundantBitwiseAndIdent(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.SimplifyRedundantLogicalExpression{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(x int) int {
				return x & x
			}
		`, `
			package main

			func f(x int) int {
				return x
			}
		`),
	)
}

func TestSimplifyRedundantBitwiseOrIdent(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.SimplifyRedundantLogicalExpression{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(x int) int {
				return x | x
			}
		`, `
			package main

			func f(x int) int {
				return x
			}
		`),
	)
}

func TestSimplifyRedundantLogicalExpressionComplex(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.SimplifyRedundantLogicalExpression{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(a, b bool) {
				if a && b && a && b {
					println("yes")
				}
			}
		`),
	)
}

func TestSimplifyRedundantLogicalExpressionNoChangeDifferent(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.SimplifyRedundantLogicalExpression{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(x, y bool) {
				if x && y {
					println("both")
				}
			}
		`),
	)
}

func TestSimplifyRedundantLogicalExpressionNoChangeArithmetic(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.SimplifyRedundantLogicalExpression{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(x int) int {
				return x - x
			}
		`),
	)
}

func TestSimplifyRedundantLogicalExpressionNoChangeComparison(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.SimplifyRedundantLogicalExpression{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(x int) bool {
				return x == x
			}
		`),
	)
}
