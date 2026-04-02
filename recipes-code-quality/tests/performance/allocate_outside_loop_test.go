/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package performance_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/performance"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestFindNewInForLoop(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&performance.AllocateOutsideLoop{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "bytes"

			func f() {
				for i := 0; i < 10; i++ {
					buf := new(bytes.Buffer)
					_ = buf
				}
			}
		`),
	)
}

func TestFindNewInRangeLoop(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&performance.AllocateOutsideLoop{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(items []string) {
				for range items {
					n := new(int)
					_ = n
				}
			}
		`),
	)
}

func TestFindNewNoChangeOutsideLoop(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&performance.AllocateOutsideLoop{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "bytes"

			func f() {
				buf := new(bytes.Buffer)
				_ = buf
			}
		`),
	)
}
