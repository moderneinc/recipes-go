/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/simplification"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestSimplifyTrimLeftNoop(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.SimplifyTrimLeftNoop{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "strings"

			func f(s string) string {
				return strings.TrimLeft(s, "")
			}
		`, `
			package main

			import "strings"

			func f(s string) string {
				return s
			}
		`),
	)
}

func TestSimplifyTrimRightNoop(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.SimplifyTrimLeftNoop{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "strings"

			func f(s string) string {
				return strings.TrimRight(s, "")
			}
		`, `
			package main

			import "strings"

			func f(s string) string {
				return s
			}
		`),
	)
}

func TestSimplifyTrimLeftNoopNoChange(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.SimplifyTrimLeftNoop{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "strings"

			func f(s string) string {
				return strings.TrimLeft(s, " ")
			}
		`),
	)
}
