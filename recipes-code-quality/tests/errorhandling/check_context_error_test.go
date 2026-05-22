/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes-code-quality/recipes/errorhandling"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestCheckContextError(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.CheckContextError{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "context"

			func f(ctx context.Context) error {
				return ctx.Err()
			}
		`, `
			package main

			import "context"

			func f(ctx context.Context) error {
				return/*~~(ctx.Err() found; inspect the context error)~~>*/ ctx.Err()
			}
		`),
	)
}

func TestCheckContextErrorNoChangeDone(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.CheckContextError{})
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
