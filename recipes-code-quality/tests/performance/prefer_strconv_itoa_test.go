/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package performance_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/performance"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestPreferStrconvItoaSimple(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&performance.PreferStrconvItoa{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "fmt"

			func f(n int) string {
				return fmt.Sprintf("%d", n)
			}
		`, `
			package main

			import "fmt"

			func f(n int) string {
				return strconv.Itoa(n)
			}
		`),
	)
}

func TestPreferStrconvItoaNoChangeOtherFormat(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&performance.PreferStrconvItoa{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "fmt"

			func f(n int) string {
				return fmt.Sprintf("%04d", n)
			}
		`),
	)
}

func TestPreferStrconvItoaNoChangeString(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&performance.PreferStrconvItoa{})
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
