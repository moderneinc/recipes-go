/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/simplification"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestPreferStringsContainsRunePositive(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.PreferStringsContainsRune{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "strings"

			func f(s string) bool {
				return strings.IndexRune(s, 'a') != -1
			}
		`, `
			package main

			import "strings"

			func f(s string) bool {
				return strings.ContainsRune(s, 'a')
			}
		`),
	)
}

func TestPreferStringsContainsRuneGte0(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.PreferStringsContainsRune{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "strings"

			func f(s string) bool {
				return strings.IndexRune(s, 'a') >= 0
			}
		`, `
			package main

			import "strings"

			func f(s string) bool {
				return strings.ContainsRune(s, 'a')
			}
		`),
	)
}

func TestPreferStringsContainsRuneNegative(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.PreferStringsContainsRune{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "strings"

			func f(s string) bool {
				return strings.IndexRune(s, 'a') == -1
			}
		`, `
			package main

			import "strings"

			func f(s string) bool {
				return !strings.ContainsRune(s, 'a')
			}
		`),
	)
}

func TestPreferStringsContainsRuneLt0(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.PreferStringsContainsRune{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "strings"

			func f(s string) bool {
				return strings.IndexRune(s, 'a') < 0
			}
		`, `
			package main

			import "strings"

			func f(s string) bool {
				return !strings.ContainsRune(s, 'a')
			}
		`),
	)
}

func TestPreferStringsContainsRuneNoChange(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.PreferStringsContainsRune{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "strings"

			func f(s string) int {
				return strings.IndexRune(s, 'a')
			}
		`),
	)
}
