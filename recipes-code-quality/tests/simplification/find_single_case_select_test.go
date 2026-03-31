/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/simplification"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestFindSingleCaseSelect(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.FindSingleCaseSelect{})
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

func TestFindSingleCaseSelectNoChangeWithDefault(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.FindSingleCaseSelect{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(ch chan int) {
				select {
				case v := <-ch:
					println(v)
				default:
					println("no value")
				}
			}
		`),
	)
}

func TestFindSingleCaseSelectNoChangeMultipleCases(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.FindSingleCaseSelect{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(ch1, ch2 chan int) {
				select {
				case v := <-ch1:
					println(v)
				case v := <-ch2:
					println(v)
				}
			}
		`),
	)
}
