/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/simplification"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestPreferStringsContainsAnyNotEqNeg1(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.PreferStringsContainsAny{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "strings"

			func f(s string) bool {
				return strings.IndexAny(s, "aeiou") != -1
			}
		`, `
			package main

			import "strings"

			func f(s string) bool {
				return strings.ContainsAny(s, "aeiou")
			}
		`),
	)
}

func TestPreferStringsContainsAnyGte0(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.PreferStringsContainsAny{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "strings"

			func f(s string) bool {
				return strings.IndexAny(s, "aeiou") >= 0
			}
		`, `
			package main

			import "strings"

			func f(s string) bool {
				return strings.ContainsAny(s, "aeiou")
			}
		`),
	)
}

func TestPreferStringsContainsAnyEqNeg1(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.PreferStringsContainsAny{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "strings"

			func f(s string) bool {
				return strings.IndexAny(s, "aeiou") == -1
			}
		`, `
			package main

			import "strings"

			func f(s string) bool {
				return !strings.ContainsAny(s, "aeiou")
			}
		`),
	)
}

func TestPreferStringsContainsAnyLt0(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.PreferStringsContainsAny{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "strings"

			func f(s string) bool {
				return strings.IndexAny(s, "aeiou") < 0
			}
		`, `
			package main

			import "strings"

			func f(s string) bool {
				return !strings.ContainsAny(s, "aeiou")
			}
		`),
	)
}

func TestPreferStringsContainsAnyNoChange(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.PreferStringsContainsAny{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "strings"

			func f(s string) int {
				return strings.IndexAny(s, "aeiou")
			}
		`),
	)
}
