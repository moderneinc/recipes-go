/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package performance_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/performance"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestPreferStrconvFormatBoolSimple(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&performance.PreferStrconvFormatBool{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "fmt"

			func f(b bool) string {
				return fmt.Sprintf("%t", b)
			}
		`, `
			package main

			import "fmt"

			func f(b bool) string {
				return strconv.FormatBool(b)
			}
		`),
	)
}

func TestPreferStrconvFormatBoolNoChangeOtherFormat(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&performance.PreferStrconvFormatBool{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "fmt"

			func f(n int) string {
				return fmt.Sprintf("%d", n)
			}
		`),
	)
}

func TestPreferStrconvFormatBoolNoChangeMultipleArgs(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&performance.PreferStrconvFormatBool{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "fmt"

			func f(b bool) string {
				return fmt.Sprintf("value: %t", b)
			}
		`),
	)
}
