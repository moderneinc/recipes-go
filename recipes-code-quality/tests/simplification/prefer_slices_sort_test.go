/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/simplification"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestPreferSlicesSortInts(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.PreferSlicesSort{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "sort"

			func f(s []int) {
				sort.Ints(s)
			}
		`, `
			package main

			import "sort"

			func f(s []int) {
				slices.Sort(s)
			}
		`),
	)
}

func TestPreferSlicesSortStrings(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.PreferSlicesSort{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "sort"

			func f(s []string) {
				sort.Strings(s)
			}
		`, `
			package main

			import "sort"

			func f(s []string) {
				slices.Sort(s)
			}
		`),
	)
}

func TestPreferSlicesSortFloat64s(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.PreferSlicesSort{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "sort"

			func f(s []float64) {
				sort.Float64s(s)
			}
		`, `
			package main

			import "sort"

			func f(s []float64) {
				slices.Sort(s)
			}
		`),
	)
}

func TestPreferSlicesSortNoChange(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.PreferSlicesSort{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "sort"

			func f(s []int) {
				sort.Slice(s, func(i, j int) bool { return s[i] < s[j] })
			}
		`),
	)
}
