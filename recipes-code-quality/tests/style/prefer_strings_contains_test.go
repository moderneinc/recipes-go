/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/style"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestPreferStringsContainsNotEqualNegOne(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.PreferStringsContains{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "strings"

			func f(s string) bool {
				return strings.Index(s, "foo") != -1
			}
		`, `
			package main

			import "strings"

			func f(s string) bool {
				return strings.Contains(s, "foo")
			}
		`),
	)
}

func TestPreferStringsContainsEqualNegOne(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.PreferStringsContains{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "strings"

			func f(s string) bool {
				return strings.Index(s, "foo") == -1
			}
		`, `
			package main

			import "strings"

			func f(s string) bool {
				return !strings.Contains(s, "foo")
			}
		`),
	)
}

func TestPreferStringsContainsNoChangeGreaterThanZero(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.PreferStringsContains{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "strings"

			func f(s string) bool {
				return strings.Index(s, "foo") > 0
			}
		`),
	)
}
