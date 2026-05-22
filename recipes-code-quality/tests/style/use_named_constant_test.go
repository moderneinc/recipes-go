/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes-code-quality/recipes/style"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestUseNamedConstant(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.UseNamedConstant{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() int {
				return 42
			}
		`, `
			package main

			func f() int {
				return /*~~(magic number; consider using a named constant)~~>*/42
			}
		`),
	)
}

func TestUseNamedConstantNoChangeZero(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.UseNamedConstant{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() int {
				return 0
			}
		`),
	)
}

func TestUseNamedConstantNoChangeOne(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.UseNamedConstant{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() int {
				return 1
			}
		`),
	)
}

func TestUseNamedConstantNoChangeConst(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.UseNamedConstant{})
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
