/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/simplification"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestPreferStringsContainsOverCountPositive(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.PreferStringsContainsOverCount{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "strings"

			func f(s string) bool {
				return strings.Count(s, "sub") > 0
			}
		`, `
			package main

			import "strings"

			func f(s string) bool {
				return strings.Contains(s, "sub")
			}
		`),
	)
}

func TestPreferStringsContainsOverCountNegative(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.PreferStringsContainsOverCount{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "strings"

			func f(s string) bool {
				return strings.Count(s, "sub") == 0
			}
		`, `
			package main

			import "strings"

			func f(s string) bool {
				return !strings.Contains(s, "sub")
			}
		`),
	)
}

func TestPreferStringsContainsOverCountNoChange(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.PreferStringsContainsOverCount{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "strings"

			func f(s string) int {
				return strings.Count(s, "sub")
			}
		`),
	)
}
