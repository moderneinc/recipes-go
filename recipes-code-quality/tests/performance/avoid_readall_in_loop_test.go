/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package performance_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/performance"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestFindReadAllInForLoop(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&performance.AvoidReadAllInLoop{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "io"

			func f(r io.Reader) {
				for i := 0; i < 10; i++ {
					_, _ = io.ReadAll(r)
				}
			}
		`),
	)
}

func TestFindReadAllNoChangeOutsideLoop(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&performance.AvoidReadAllInLoop{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "io"

			func f(r io.Reader) {
				_, _ = io.ReadAll(r)
			}
		`),
	)
}
