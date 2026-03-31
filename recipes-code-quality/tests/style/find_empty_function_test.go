/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/style"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestFindEmptyFunction(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindEmptyFunction{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() {}
		`),
	)
}

func TestFindEmptyFunctionNoChangeWithBody(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindEmptyFunction{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() {
				return
			}
		`),
	)
}
