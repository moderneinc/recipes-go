/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes-code-quality/recipes/style"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestAvoidGlobalVariable(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.AvoidGlobalVariable{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			var x = 1
		`, `
			package main

			/*~~(avoid global variable)~~>*/var x = 1
		`),
	)
}

func TestAvoidGlobalVariableNoChangeConst(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.AvoidGlobalVariable{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			const x = 1
		`),
	)
}

func TestAvoidGlobalVariableNoChangeLocalVar(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.AvoidGlobalVariable{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() {
				var x = 1
				_ = x
			}
		`),
	)
}
