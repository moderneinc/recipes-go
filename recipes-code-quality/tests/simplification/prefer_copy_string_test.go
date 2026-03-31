/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/simplification"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestPreferCopyString(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.PreferCopyString{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(s string) int {
				dst := make([]byte, len(s))
				return copy(dst, []byte(s))
			}
		`, `
			package main

			func f(s string) int {
				dst := make([]byte, len(s))
				return copy(dst, s)
			}
		`),
	)
}

func TestPreferCopyStringNoChange(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.PreferCopyString{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(s string) int {
				dst := make([]byte, len(s))
				return copy(dst, s)
			}
		`),
	)
}
