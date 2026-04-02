/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/style"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestLimitFunctionParameters(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.LimitFunctionParameters{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(a int, b int, c int, d int, e int, g int) {
				_ = a + b + c + d + e + g
			}
		`),
	)
}

func TestLimitFunctionParametersNoChangeFew(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.LimitFunctionParameters{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(a int, b int) {
				_ = a + b
			}
		`),
	)
}
