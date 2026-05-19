/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes-code-quality/recipes/simplification"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestSimplifySplitN(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.SimplifySplitN{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "strings"

			func f(s string) []string {
				return strings.SplitN(s, ",", -1)
			}
		`, `
			package main

			import "strings"

			func f(s string) []string {
				return strings.Split(s, ",")
			}
		`),
	)
}

func TestSimplifySplitNNoChange(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.SimplifySplitN{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "strings"

			func f(s string) []string {
				return strings.SplitN(s, ",", 2)
			}
		`),
	)
}
