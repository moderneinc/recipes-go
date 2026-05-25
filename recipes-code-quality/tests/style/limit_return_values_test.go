/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes-code-quality/recipes/style"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestLimitReturnValues(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.LimitReturnValues{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() (int, int, int, int) {
				return 1, 2, 3, 4
			}
		`, `
			package main

			func /*~~(function has too many return values)~~>*/f() (int, int, int, int) {
				return 1, 2, 3, 4
			}
		`),
	)
}

func TestLimitReturnValuesNoChangeFew(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.LimitReturnValues{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() (int, int) {
				return 1, 2
			}
		`),
	)
}
