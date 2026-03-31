/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/errorhandling"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestFindContextErr(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.FindContextErr{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "context"

			func f(ctx context.Context) error {
				return ctx.Err()
			}
		`),
	)
}

func TestFindContextErrNoChangeDone(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.FindContextErr{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "context"

			func f(ctx context.Context) <-chan struct{} {
				return ctx.Done()
			}
		`),
	)
}
