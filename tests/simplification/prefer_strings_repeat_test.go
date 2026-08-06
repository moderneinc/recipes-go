/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes/simplification"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestSimplifySprintfConcat(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.SimplifySprintfConcat{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "fmt"

			func f(a, b string) string {
				return fmt.Sprintf("%s%s", a, b)
			}
		`, `
			package main

			func f(a, b string) string {
				return a + b
			}
		`),
	)
}

func TestSimplifySprintfConcatNoChangeFormat(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.SimplifySprintfConcat{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "fmt"

			func f(a, b string) string {
				return fmt.Sprintf("%s-%s", a, b)
			}
		`),
	)
}

// Skips []byte arguments, since %s accepts them but []byte values cannot be added.
func TestSimplifySprintfConcatNoChangeBytes(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.SimplifySprintfConcat{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "fmt"

			func f(a, b []byte) string {
				return fmt.Sprintf("%s%s", a, b)
			}
		`),
	)
}
