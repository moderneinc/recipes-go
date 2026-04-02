/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package redundancy_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/redundancy"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestSimplifyRedundantLenBeforeRangeSimple(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.SimplifyRedundantLenBeforeRange{})
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
		`, `
			package main

			func f(s []int) {
				for _, v := range s {
					println(v)
				}
			}
		`),
	)
}

func TestSimplifyRedundantLenBeforeRangeNotEqual(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.SimplifyRedundantLenBeforeRange{})
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
		`, `
			package main

			func f(s []string) {
				for _, v := range s {
					println(v)
				}
			}
		`),
	)
}

func TestSimplifyRedundantLenBeforeRangeNoChangeNotRange(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.SimplifyRedundantLenBeforeRange{})
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

func TestSimplifyRedundantLenBeforeRangeNoChangeDifferentVar(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.SimplifyRedundantLenBeforeRange{})
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

func TestSimplifyRedundantLenBeforeRangeNoChangeWithElse(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.SimplifyRedundantLenBeforeRange{})
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
