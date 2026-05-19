/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package naming_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes-code-quality/recipes/naming"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestUseCtxForContextParam(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&naming.UseCtxForContextParam{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "context"

			func f(context context.Context) {
			}
		`, `
			package main

			import "context"

			func f(ctx context.Context) {
			}
		`),
	)
}

func TestUseCtxForContextParamNoChange(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&naming.UseCtxForContextParam{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "context"

			func f(ctx context.Context) {
			}
		`),
	)
}
