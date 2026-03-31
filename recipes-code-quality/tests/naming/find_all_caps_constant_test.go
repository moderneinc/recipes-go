/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package naming_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/naming"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestFindAllCapsConstant(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&naming.FindAllCapsConstant{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			const MAX_BUFFER_SIZE = 1024
		`),
	)
}

func TestFindAllCapsConstantNoChangeMixedCaps(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&naming.FindAllCapsConstant{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			const MaxBufferSize = 1024
		`),
	)
}

func TestFindAllCapsConstantNoChangeNoUnderscore(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&naming.FindAllCapsConstant{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			const EOF = 0
		`),
	)
}
