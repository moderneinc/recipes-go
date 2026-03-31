/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/style"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestFindGlobalVar(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindGlobalVar{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			var x = 1
		`),
	)
}

func TestFindGlobalVarNoChangeConst(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindGlobalVar{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			const x = 1
		`),
	)
}

func TestFindGlobalVarNoChangeLocalVar(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindGlobalVar{})
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
