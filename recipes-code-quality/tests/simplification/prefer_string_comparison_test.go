/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/simplification"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestPreferStringComparisonEq(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.PreferStringComparison{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "strings"

			func f(a, b string) bool {
				return strings.Compare(a, b) == 0
			}
		`, `
			package main

			import "strings"

			func f(a, b string) bool {
				return a == b
			}
		`),
	)
}

func TestPreferStringComparisonNeq(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.PreferStringComparison{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "strings"

			func f(a, b string) bool {
				return strings.Compare(a, b) != 0
			}
		`, `
			package main

			import "strings"

			func f(a, b string) bool {
				return a != b
			}
		`),
	)
}

func TestPreferStringComparisonLt(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.PreferStringComparison{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "strings"

			func f(a, b string) bool {
				return strings.Compare(a, b) < 0
			}
		`, `
			package main

			import "strings"

			func f(a, b string) bool {
				return a < b
			}
		`),
	)
}

func TestPreferStringComparisonGt(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.PreferStringComparison{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "strings"

			func f(a, b string) bool {
				return strings.Compare(a, b) > 0
			}
		`, `
			package main

			import "strings"

			func f(a, b string) bool {
				return a > b
			}
		`),
	)
}

func TestPreferStringComparisonNoChange(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.PreferStringComparison{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(a, b string) bool {
				return a == b
			}
		`),
	)
}
