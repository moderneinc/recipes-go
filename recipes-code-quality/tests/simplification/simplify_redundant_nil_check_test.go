/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/simplification"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestSimplifyRedundantNilCheckSlice(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.SimplifyRedundantNilCheck{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(s []int) {
				if s != nil && len(s) > 0 {
					println("has items")
				}
			}
		`, `
			package main

			func f(s []int) {
				if len(s) > 0 {
					println("has items")
				}
			}
		`),
	)
}

func TestSimplifyRedundantNilCheckNoChangeOtherOp(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.SimplifyRedundantNilCheck{})
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
