/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes-code-quality/recipes/simplification"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestPreferLenCheckGte(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.PreferLenCheck{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(s []int) bool {
				return len(s) >= 1
			}
		`, `
			package main

			func f(s []int) bool {
				return len(s) > 0
			}
		`),
	)
}

func TestPreferLenCheckLt(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.PreferLenCheck{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(s []int) bool {
				return len(s) < 1
			}
		`, `
			package main

			func f(s []int) bool {
				return len(s) == 0
			}
		`),
	)
}

func TestPreferLenCheckNoChange(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.PreferLenCheck{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(s []int) bool {
				return len(s) > 0
			}
		`),
	)
}
