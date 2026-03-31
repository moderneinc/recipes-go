/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package performance_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/performance"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestFindFmtSprintfInForLoop(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&performance.FindFmtInLoop{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "fmt"

			func f() {
				for i := 0; i < 10; i++ {
					_ = fmt.Sprintf("%d", i)
				}
			}
		`),
	)
}

func TestFindFmtSprintInRangeLoop(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&performance.FindFmtInLoop{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "fmt"

			func f(items []int) {
				for _, item := range items {
					_ = fmt.Sprint(item)
				}
			}
		`),
	)
}

func TestFindFmtNoChangeOutsideLoop(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&performance.FindFmtInLoop{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "fmt"

			func f() {
				_ = fmt.Sprintf("%d", 42)
			}
		`),
	)
}
