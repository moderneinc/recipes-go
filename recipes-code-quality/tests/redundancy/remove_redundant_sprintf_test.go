/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package redundancy_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/redundancy"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestRemoveRedundantSprintfSimple(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.RemoveRedundantSprintf{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "fmt"

			func f(s string) string {
				return fmt.Sprintf("%s", s)
			}
		`, `
			package main

			import "fmt"

			func f(s string) string {
				return s
			}
		`),
	)
}

func TestRemoveRedundantSprintfNoChangeMultipleVerbs(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.RemoveRedundantSprintf{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "fmt"

			func f(s string) string {
				return fmt.Sprintf("hello %s", s)
			}
		`),
	)
}

func TestRemoveRedundantSprintfNoChangeFormatD(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.RemoveRedundantSprintf{})
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
