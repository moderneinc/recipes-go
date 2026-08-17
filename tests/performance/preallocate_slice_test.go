/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package performance_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes/performance"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestPreallocateSliceAddsCapacity(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&performance.PreallocateSlice{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(xs []int) []int {
				out := make([]int, 0)
				for _, x := range xs {
					out = append(out, x)
				}
				return out
			}
		`, `
			package main

			func f(xs []int) []int {
				out := make([]int, 0, len(xs))
				for _, x := range xs {
					out = append(out, x)
				}
				return out
			}
		`),
	)
}

func TestPreallocateSliceAppendUnderCondition(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&performance.PreallocateSlice{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(xs []int) []int {
				out := make([]int, 0)
				for _, x := range xs {
					if x > 0 {
						out = append(out, x)
					}
				}
				return out
			}
		`, `
			package main

			func f(xs []int) []int {
				out := make([]int, 0, len(xs))
				for _, x := range xs {
					if x > 0 {
						out = append(out, x)
					}
				}
				return out
			}
		`),
	)
}

func TestPreallocateSliceKeepsExistingCapacity(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&performance.PreallocateSlice{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(xs []int) []int {
				out := make([]int, 0, 8)
				for _, x := range xs {
					out = append(out, x)
				}
				return out
			}
		`),
	)
}

func TestPreallocateSliceIgnoresOtherSlice(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&performance.PreallocateSlice{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(xs []int, other []int) []int {
				out := make([]int, 0)
				for _, x := range xs {
					other = append(other, x)
				}
				return out
			}
		`),
	)
}

// len() of a call expression would evaluate it a second time.
func TestPreallocateSliceIgnoresCallIterable(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&performance.PreallocateSlice{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func g() []int { return nil }

			func f() []int {
				out := make([]int, 0)
				for _, x := range g() {
					out = append(out, x)
				}
				return out
			}
		`),
	)
}

// The iterable is only known to be in scope at the make when the loop follows
// it directly.
func TestPreallocateSliceIgnoresSeparatedLoop(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&performance.PreallocateSlice{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() []int {
				out := make([]int, 0)
				ys := []int{1, 2}
				for _, y := range ys {
					out = append(out, y)
				}
				return out
			}
		`),
	)
}

func TestPreallocateSliceIgnoresVarDeclaration(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&performance.PreallocateSlice{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(xs []int) []int {
				var out []int
				for _, x := range xs {
					out = append(out, x)
				}
				return out
			}
		`),
	)
}
