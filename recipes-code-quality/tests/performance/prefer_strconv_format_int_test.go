/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package performance_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/performance"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestSimplifySprintfCharSimple(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&performance.SimplifySprintfChar{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "fmt"

			func f(r rune) string {
				return fmt.Sprintf("%c", r)
			}
		`, `
			package main

			import "fmt"

			func f(r rune) string {
				return string(r)
			}
		`),
	)
}

func TestSimplifySprintfCharNoChangeOtherFormat(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&performance.SimplifySprintfChar{})
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

func TestSimplifySprintfCharNoChangeMultipleArgs(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&performance.SimplifySprintfChar{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "fmt"

			func f(r rune) string {
				return fmt.Sprintf("char: %c", r)
			}
		`),
	)
}
