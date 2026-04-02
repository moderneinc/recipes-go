/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/simplification"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestSimplifyDoubleNegation(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.SimplifyDoubleNegation{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(ok bool) bool {
				return !!ok
			}
		`, `
			package main

			func f(ok bool) bool {
				return ok
			}
		`),
	)
}

func TestSimplifyDoubleNegationNoChangeSingle(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.SimplifyDoubleNegation{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(ok bool) bool {
				return !ok
			}
		`),
	)
}
