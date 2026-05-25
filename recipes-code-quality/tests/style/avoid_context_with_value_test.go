/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes-code-quality/recipes/style"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestAvoidContextWithValue(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.AvoidContextWithValue{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "context"

			type key string

			func f(ctx context.Context) {
				ctx = context.WithValue(ctx, key("k"), "val")
				_ = ctx
			}
		`, `
			package main

			import "context"

			type key string

			func f(ctx context.Context) {
				ctx =/*~~(context.WithValue() call; consider passing dependencies explicitly)~~>*/ context.WithValue(ctx, key("k"), "val")
				_ = ctx
			}
		`),
	)
}

func TestAvoidContextWithValueNoChangeWithCancel(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.AvoidContextWithValue{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "context"

			func f(ctx context.Context) {
				ctx, cancel := context.WithCancel(ctx)
				defer cancel()
				_ = ctx
			}
		`),
	)
}
