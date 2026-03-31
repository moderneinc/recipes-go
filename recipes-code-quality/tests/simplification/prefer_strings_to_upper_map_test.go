/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/simplification"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestPreferStringsToUpperMap(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.PreferStringsToUpperMap{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import (
				"strings"
				"unicode"
			)

			func f(s string) string {
				return strings.Map(unicode.ToUpper, s)
			}
		`, `
			package main

			import (
				"strings"
				"unicode"
			)

			func f(s string) string {
				return strings.ToUpper(s)
			}
		`),
	)
}

func TestPreferStringsToUpperMapNoChange(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.PreferStringsToUpperMap{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "strings"

			func f(s string) string {
				return strings.ToUpper(s)
			}
		`),
	)
}
