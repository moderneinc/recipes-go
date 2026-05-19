/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package naming_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes-code-quality/recipes/naming"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestUseErrPrefixForErrors(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&naming.UseErrPrefixForErrors{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "errors"

			var notFound = errors.New("not found")
		`, `
			package main

			import "errors"

			var ErrNotFound = errors.New("not found")
		`),
	)
}

func TestUseErrPrefixForErrorsNoChangeCorrect(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&naming.UseErrPrefixForErrors{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "errors"

			var ErrNotFound = errors.New("not found")
		`),
	)
}

func TestUseErrPrefixForErrorsNoChangeConst(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&naming.UseErrPrefixForErrors{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			const x = 1
		`),
	)
}
