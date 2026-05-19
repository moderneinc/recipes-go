/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes-code-quality/recipes/simplification"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestPreferEmptyStringCheckEqual(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.PreferEmptyStringCheck{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(s string) bool {
				return len(s) == 0
			}
		`, `
			package main

			func f(s string) bool {
				return s == ""
			}
		`),
	)
}

func TestPreferEmptyStringCheckNotEqual(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.PreferEmptyStringCheck{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(s string) bool {
				return len(s) != 0
			}
		`, `
			package main

			func f(s string) bool {
				return s != ""
			}
		`),
	)
}
