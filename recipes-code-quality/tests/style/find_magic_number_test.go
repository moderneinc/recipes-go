/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/style"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestFindMagicNumber(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindMagicNumber{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() int {
				return 42
			}
		`),
	)
}

func TestFindMagicNumberNoChangeZero(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindMagicNumber{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() int {
				return 0
			}
		`),
	)
}

func TestFindMagicNumberNoChangeOne(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindMagicNumber{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() int {
				return 1
			}
		`),
	)
}

func TestFindMagicNumberNoChangeConst(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindMagicNumber{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			const maxRetries = 5

			func f() int {
				return 0
			}
		`),
	)
}
