/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package redundancy_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/redundancy"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestRemoveSelfAssignmentSimple(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.RemoveSelfAssignment{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() {
				x := 1
				x = x
				_ = x
			}
		`, "package main\n\nfunc f() {\n\tx := 1\n\t\n\t_ = x\n}"),
	)
}

func TestRemoveSelfAssignmentNoChangeDifferentNames(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.RemoveSelfAssignment{})
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

func TestRemoveSelfAssignmentNoChangeExpression(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.RemoveSelfAssignment{})
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
