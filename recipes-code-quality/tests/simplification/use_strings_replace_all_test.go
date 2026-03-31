/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/simplification"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestUseStringsReplaceAll(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.UseStringsReplaceAll{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "strings"

			func f(s string) string {
				return strings.Replace(s, "old", "new", -1)
			}
		`, `
			package main

			import "strings"

			func f(s string) string {
				return strings.ReplaceAll(s, "old", "new")
			}
		`),
	)
}

func TestUseStringsReplaceAllNoChange(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.UseStringsReplaceAll{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "strings"

			func f(s string) string {
				return strings.Replace(s, "old", "new", 1)
			}
		`),
	)
}
