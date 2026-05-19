/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes-code-quality/recipes/style"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestPreferStringsEqualFoldToLower(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.PreferStringsEqualFold{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "strings"

			func f(a, b string) bool {
				return strings.ToLower(a) == strings.ToLower(b)
			}
		`, `
			package main

			import "strings"

			func f(a, b string) bool {
				return strings.EqualFold(a, b)
			}
		`),
	)
}

func TestPreferStringsEqualFoldToUpper(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.PreferStringsEqualFold{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "strings"

			func f(a, b string) bool {
				return strings.ToUpper(a) == strings.ToUpper(b)
			}
		`, `
			package main

			import "strings"

			func f(a, b string) bool {
				return strings.EqualFold(a, b)
			}
		`),
	)
}

func TestPreferStringsEqualFoldNoChange(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.PreferStringsEqualFold{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(a, b string) bool {
				return a == b
			}
		`),
	)
}
