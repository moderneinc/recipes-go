/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package redundancy_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/redundancy"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestFindUnreachableCodeSimple(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.FindUnreachableCode{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() int {
				return 1
				println("unreachable")
			}
		`),
	)
}

func TestFindUnreachableCodeMultipleStatements(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.FindUnreachableCode{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() int {
				return 1
				x := 2
				println(x)
			}
		`),
	)
}

func TestFindUnreachableCodeNoChangeNoReturn(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.FindUnreachableCode{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() {
				println("hello")
				println("world")
			}
		`),
	)
}

func TestFindUnreachableCodeNoChangeReturnAtEnd(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.FindUnreachableCode{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() int {
				x := 1
				return x
			}
		`),
	)
}
