/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/style"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestPreferStrconvQuote(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.PreferStrconvQuote{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "fmt"

			func f(s string) string {
				return fmt.Sprintf("%q", s)
			}
		`, `
			package main

			import "fmt"

			func f(s string) string {
				return strconv.Quote(s)
			}
		`),
	)
}

func TestPreferStrconvQuoteNoChangeOtherVerb(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.PreferStrconvQuote{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "fmt"

			func f(s string) string {
				return fmt.Sprintf("%s", s)
			}
		`),
	)
}
