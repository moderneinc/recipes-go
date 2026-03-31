/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package naming_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/naming"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestFindSingleLetterVar(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&naming.FindSingleLetterVar{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			var q = 1
		`),
	)
}

func TestFindSingleLetterVarNoChangeConventional(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&naming.FindSingleLetterVar{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			var i = 0
		`),
	)
}

func TestFindSingleLetterVarNoChangeLongerName(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&naming.FindSingleLetterVar{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			var count = 10
		`),
	)
}
