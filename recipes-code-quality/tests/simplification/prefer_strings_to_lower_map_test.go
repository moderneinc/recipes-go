/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes-code-quality/recipes/simplification"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestPreferStringsToLowerMap(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.PreferStringsToLowerMap{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import (
				"strings"
				"unicode"
			)

			func f(s string) string {
				return strings.Map(unicode.ToLower, s)
			}
		`, `
			package main

			import (
				"strings"
				"unicode"
			)

			func f(s string) string {
				return strings.ToLower(s)
			}
		`),
	)
}

func TestPreferStringsToLowerMapNoChange(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.PreferStringsToLowerMap{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "strings"

			func f(s string) string {
				return strings.ToLower(s)
			}
		`),
	)
}
