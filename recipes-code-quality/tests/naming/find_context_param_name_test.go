/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package naming_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/naming"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestFindContextParamNotCtx(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&naming.FindContextParamNotCtx{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "context"

			func f(context context.Context) {
			}
		`),
	)
}

func TestFindContextParamNotCtxNoChange(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&naming.FindContextParamNotCtx{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "context"

			func f(ctx context.Context) {
			}
		`),
	)
}
