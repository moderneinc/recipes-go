/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes-code-quality/recipes/simplification"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestRemoveRedundantRangeBlank(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.RemoveRedundantRangeBlank{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(s []int) {
				for i, _ := range s {
					println(i)
				}
			}
		`, `
			package main

			func f(s []int) {
				for i := range s {
					println(i)
				}
			}
		`),
	)
}

func TestRemoveRedundantRangeBlankNoChangeWithValue(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.RemoveRedundantRangeBlank{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(s []int) {
				for i, v := range s {
					println(i, v)
				}
			}
		`),
	)
}

func TestRemoveRedundantRangeBlankNoChangeKeyOnly(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.RemoveRedundantRangeBlank{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(s []int) {
				for i := range s {
					println(i)
				}
			}
		`),
	)
}
