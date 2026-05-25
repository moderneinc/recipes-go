/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package performance_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes-code-quality/recipes/performance"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestFindChannelCreateInForLoop(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&performance.CreateChannelOutsideLoop{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() {
				for i := 0; i < 10; i++ {
					ch := make(chan int)
					_ = ch
				}
			}
		`, `
			package main

			func f() {
				for i := 0; i < 10; i++ {
					ch :=/*~~(channel creation in loop; consider creating the channel once before the loop)~~>*/ make(chan int)
					_ = ch
				}
			}
		`),
	)
}

func TestFindChannelCreateInRangeLoop(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&performance.CreateChannelOutsideLoop{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(items []string) {
				for range items {
					ch := make(chan string)
					_ = ch
				}
			}
		`, `
			package main

			func f(items []string) {
				for range items {
					ch :=/*~~(channel creation in loop; consider creating the channel once before the loop)~~>*/ make(chan string)
					_ = ch
				}
			}
		`),
	)
}

func TestFindChannelCreateNoChangeOutsideLoop(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&performance.CreateChannelOutsideLoop{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() {
				ch := make(chan int)
				_ = ch
			}
		`),
	)
}
