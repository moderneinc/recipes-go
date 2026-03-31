/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package naming_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/naming"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestFindMisnamedErrorVar(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&naming.FindMisnamedErrorVar{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "errors"

			var notFound = errors.New("not found")
		`),
	)
}

func TestFindMisnamedErrorVarNoChangeCorrect(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&naming.FindMisnamedErrorVar{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "errors"

			var ErrNotFound = errors.New("not found")
		`),
	)
}

func TestFindMisnamedErrorVarNoChangeConst(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&naming.FindMisnamedErrorVar{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			const x = 1
		`),
	)
}
