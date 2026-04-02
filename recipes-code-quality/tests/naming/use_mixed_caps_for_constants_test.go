/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package naming_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/naming"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestUseMixedCapsForConstants(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&naming.UseMixedCapsForConstants{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			const MAX_BUFFER_SIZE = 1024
		`, `
			package main

			const MaxBufferSize = 1024
		`),
	)
}

func TestUseMixedCapsForConstantsNoChangeMixedCaps(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&naming.UseMixedCapsForConstants{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			const MaxBufferSize = 1024
		`),
	)
}

func TestUseMixedCapsForConstantsNoChangeNoUnderscore(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&naming.UseMixedCapsForConstants{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			const EOF = 0
		`),
	)
}
