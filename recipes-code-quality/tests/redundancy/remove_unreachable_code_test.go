/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package redundancy_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/redundancy"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestRemoveUnreachableCodeSimple(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.RemoveUnreachableCode{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() int {
				return 1
				println("unreachable")
			}
		`, `
			package main

			func f() int {
				return 1
			}
		`),
	)
}

func TestRemoveUnreachableCodeMultipleStatements(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.RemoveUnreachableCode{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() int {
				return 1
				x := 2
				println(x)
			}
		`, `
			package main

			func f() int {
				return 1
			}
		`),
	)
}

func TestRemoveUnreachableCodeNoChangeNoReturn(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.RemoveUnreachableCode{})
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

func TestRemoveUnreachableCodeNoChangeReturnAtEnd(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.RemoveUnreachableCode{})
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
