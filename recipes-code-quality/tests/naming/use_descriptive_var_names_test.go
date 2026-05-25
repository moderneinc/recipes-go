/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package naming_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes-code-quality/recipes/naming"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestUseDescriptiveVarNames(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&naming.UseDescriptiveVarNames{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			var q = 1
		`, `
			package main

			var /*~~(single-letter variable name is not a conventional short name)~~>*/q = 1
		`),
	)
}

func TestUseDescriptiveVarNamesNoChangeConventional(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&naming.UseDescriptiveVarNames{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			var i = 0
		`),
	)
}

func TestUseDescriptiveVarNamesNoChangeLongerName(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&naming.UseDescriptiveVarNames{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			var count = 10
		`),
	)
}
