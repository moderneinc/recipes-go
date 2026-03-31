/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/style"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestFindManyReturns(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindManyReturns{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() (int, int, int, int) {
				return 1, 2, 3, 4
			}
		`),
	)
}

func TestFindManyReturnsNoChangeFew(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindManyReturns{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() (int, int) {
				return 1, 2
			}
		`),
	)
}
