/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes-code-quality/recipes/simplification"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestPreferStringsHasPrefixPositive(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.PreferStringsHasPrefix{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "strings"

			func f(s string) bool {
				return strings.Index(s, "foo") == 0
			}
		`, `
			package main

			import "strings"

			func f(s string) bool {
				return strings.HasPrefix(s, "foo")
			}
		`),
	)
}

func TestPreferStringsHasPrefixNegative(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.PreferStringsHasPrefix{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "strings"

			func f(s string) bool {
				return strings.Index(s, "foo") != 0
			}
		`, `
			package main

			import "strings"

			func f(s string) bool {
				return !strings.HasPrefix(s, "foo")
			}
		`),
	)
}

func TestPreferStringsHasPrefixNoChange(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.PreferStringsHasPrefix{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "strings"

			func f(s string) int {
				return strings.Index(s, "foo")
			}
		`),
	)
}
