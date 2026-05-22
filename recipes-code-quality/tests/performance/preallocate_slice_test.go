/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package performance_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes-code-quality/recipes/performance"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestPreallocateSliceAppendInForLoop(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&performance.PreallocateSlice{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func main() {
				var result []int
				for i := 0; i < 10; i++ {
					result = append(result, i)
				}
			}
		`, `
			package main

			func main() {
				var result []int
				for i := 0; i < 10; i++ {
					result =/*~~(consider preallocating slice)~~>*/ append(result, i)
				}
			}
		`),
	)
}

func TestPreallocateSliceAppendInRangeLoop(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&performance.PreallocateSlice{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func main() {
				items := []string{"a", "b", "c"}
				var result []string
				for _, item := range items {
					result = append(result, item)
				}
			}
		`, `
			package main

			func main() {
				items := []string{"a", "b", "c"}
				var result []string
				for _, item := range items {
					result =/*~~(consider preallocating slice)~~>*/ append(result, item)
				}
			}
		`),
	)
}

func TestPreallocateSliceNoChangeOutsideLoop(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&performance.PreallocateSlice{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func main() {
				var result []int
				result = append(result, 1)
			}
		`),
	)
}
