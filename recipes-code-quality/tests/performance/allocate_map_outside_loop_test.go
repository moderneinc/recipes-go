/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package performance_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes-code-quality/recipes/performance"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestHoistMapAllocFromRangeLoop(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&performance.AllocateMapOutsideLoop{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(items []string) {
				for _, item := range items {
					m := make(map[string]int)
					m[item] = 1
					_ = m
				}
			}
		`, `
			package main

			func f(items []string) {
				var m = make(map[string]int)
				for _, item := range items {
					clear(m)
					m[item] = 1
					_ = m
				}
			}
		`),
	)
}

func TestHoistMapAllocFromClassicForLoop(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&performance.AllocateMapOutsideLoop{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() {
				for i := 0; i < 10; i++ {
					m := make(map[int]string)
					_ = m
				}
			}
		`, `
			package main

			func f() {
				var m = make(map[int]string)
				for i := 0; i < 10; i++ {
					clear(m)
					_ = m
				}
			}
		`),
	)
}

func TestMapAllocNoChangeOutsideLoop(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&performance.AllocateMapOutsideLoop{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() {
				m := make(map[string]int)
				m["a"] = 1
			}
		`),
	)
}
