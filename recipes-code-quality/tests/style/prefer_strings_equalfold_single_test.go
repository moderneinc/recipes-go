/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/style"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestPreferStringsEqualFoldSingleToLowerLeft(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.PreferStringsEqualFoldSingle{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "strings"

			func f(s string) bool {
				return strings.ToLower(s) == "hello"
			}
		`, `
			package main

			import "strings"

			func f(s string) bool {
				return strings.EqualFold(s, "hello")
			}
		`),
	)
}

func TestPreferStringsEqualFoldSingleToLowerRight(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.PreferStringsEqualFoldSingle{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "strings"

			func f(s string) bool {
				return "hello" == strings.ToLower(s)
			}
		`, `
			package main

			import "strings"

			func f(s string) bool {
				return strings.EqualFold(s, "hello")
			}
		`),
	)
}

func TestPreferStringsEqualFoldSingleToUpperLeft(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.PreferStringsEqualFoldSingle{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "strings"

			func f(s string) bool {
				return strings.ToUpper(s) == "HELLO"
			}
		`, `
			package main

			import "strings"

			func f(s string) bool {
				return strings.EqualFold(s, "HELLO")
			}
		`),
	)
}

func TestPreferStringsEqualFoldSingleNoChange(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.PreferStringsEqualFoldSingle{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(a, b string) bool {
				return a == b
			}
		`),
	)
}
