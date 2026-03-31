/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package redundancy_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/redundancy"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestRemoveRedundantBreakInSelectSimple(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.RemoveRedundantBreakInSelect{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(ch chan int) {
				select {
				case v := <-ch:
					println(v)
					break
				}
			}
		`, `
			package main

			func f(ch chan int) {
				select {
				case v := <-ch:
					println(v)
				}
			}
		`),
	)
}

func TestRemoveRedundantBreakInSelectMultipleCases(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.RemoveRedundantBreakInSelect{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(ch1, ch2 chan int) {
				select {
				case v := <-ch1:
					println(v)
					break
				case ch2 <- 1:
					println("sent")
					break
				}
			}
		`, `
			package main

			func f(ch1, ch2 chan int) {
				select {
				case v := <-ch1:
					println(v)
				case ch2 <- 1:
					println("sent")
				}
			}
		`),
	)
}

func TestRemoveRedundantBreakInSelectNoChange(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.RemoveRedundantBreakInSelect{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(ch chan int) {
				select {
				case v := <-ch:
					println(v)
				}
			}
		`),
	)
}

func TestRemoveRedundantBreakInSelectDefault(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.RemoveRedundantBreakInSelect{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(ch chan int) {
				select {
				case v := <-ch:
					println(v)
					break
				default:
					println("no data")
					break
				}
			}
		`, `
			package main

			func f(ch chan int) {
				select {
				case v := <-ch:
					println(v)
				default:
					println("no data")
				}
			}
		`),
	)
}
