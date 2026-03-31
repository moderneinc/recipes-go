/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package performance_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/performance"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestFindRegexMustCompileInForLoop(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&performance.FindRegexInLoop{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "regexp"

			func f(patterns []string) {
				for _, p := range patterns {
					re := regexp.MustCompile(p)
					_ = re
				}
			}
		`),
	)
}

func TestFindRegexCompileInClassicForLoop(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&performance.FindRegexInLoop{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "regexp"

			func f() {
				for i := 0; i < 10; i++ {
					re, _ := regexp.Compile("\\d+")
					_ = re
				}
			}
		`),
	)
}

func TestFindRegexNoChangeOutsideLoop(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&performance.FindRegexInLoop{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "regexp"

			var re = regexp.MustCompile("\\d+")

			func f() {
				r, _ := regexp.Compile("[a-z]+")
				_ = r
			}
		`),
	)
}
