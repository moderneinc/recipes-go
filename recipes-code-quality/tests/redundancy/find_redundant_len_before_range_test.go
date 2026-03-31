/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package redundancy_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/redundancy"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestFindRedundantLenBeforeRangeSimple(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.FindRedundantLenBeforeRange{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(s []int) {
				if len(s) > 0 {
					for _, v := range s {
						println(v)
					}
				}
			}
		`),
	)
}

func TestFindRedundantLenBeforeRangeNotEqual(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.FindRedundantLenBeforeRange{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(s []string) {
				if len(s) != 0 {
					for _, v := range s {
						println(v)
					}
				}
			}
		`),
	)
}

func TestFindRedundantLenBeforeRangeNoChangeNotRange(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.FindRedundantLenBeforeRange{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(s []int) {
				if len(s) > 0 {
					doWork(s)
				}
			}

			func doWork(s []int) {}
		`),
	)
}

func TestFindRedundantLenBeforeRangeNoChangeDifferentVar(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.FindRedundantLenBeforeRange{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(s []int, t []int) {
				if len(s) > 0 {
					for _, v := range t {
						println(v)
					}
				}
			}
		`),
	)
}

func TestFindRedundantLenBeforeRangeNoChangeWithElse(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.FindRedundantLenBeforeRange{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(s []int) {
				if len(s) > 0 {
					for _, v := range s {
						println(v)
					}
				} else {
					println("empty")
				}
			}
		`),
	)
}
