/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package redundancy_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/redundancy"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestFindSelfAssignmentSimple(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.FindSelfAssignment{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() {
				x := 1
				x = x
				_ = x
			}
		`),
	)
}

func TestFindSelfAssignmentNoChangeDifferentNames(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.FindSelfAssignment{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() {
				x := 1
				y := x
				_ = y
			}
		`),
	)
}

func TestFindSelfAssignmentNoChangeExpression(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.FindSelfAssignment{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() {
				x := 1
				x = x + 1
				_ = x
			}
		`),
	)
}
