/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes-code-quality/recipes/simplification"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestSimplifySliceRangeBasic(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.SimplifySliceRange{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(s []int) []int {
				return s[0:len(s)]
			}
		`, `
			package main

			func f(s []int) []int {
				return s[:]
			}
		`),
	)
}

func TestSimplifySliceRangeNoChangeNonZeroLow(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.SimplifySliceRange{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(s []int) []int {
				return s[1:len(s)]
			}
		`),
	)
}

func TestSimplifySliceRangeNoChangePartialHigh(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.SimplifySliceRange{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(s []int) []int {
				return s[0 : len(s)-1]
			}
		`),
	)
}

func TestSimplifySliceRangeNoChangeDifferentVar(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.SimplifySliceRange{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(s, t []int) []int {
				return s[0:len(t)]
			}
		`),
	)
}

func TestSimplifySliceRangeNoChangeThreeIndex(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.SimplifySliceRange{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(s []int) []int {
				return s[0:len(s):cap(s)]
			}
		`),
	)
}
