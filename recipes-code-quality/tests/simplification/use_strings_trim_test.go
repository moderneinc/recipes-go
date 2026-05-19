/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes-code-quality/recipes/simplification"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestSimplifyRedundantTrimSpace(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.SimplifyRedundantTrimSpace{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "strings"

			func f(s string) string {
				return strings.TrimSpace(strings.TrimSpace(s))
			}
		`, `
			package main

			import "strings"

			func f(s string) string {
				return strings.TrimSpace(s)
			}
		`),
	)
}

func TestSimplifyRedundantTrimSpaceNoChange(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.SimplifyRedundantTrimSpace{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "strings"

			func f(s string) string {
				return strings.TrimSpace(s)
			}
		`),
	)
}
