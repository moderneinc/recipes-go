/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package performance_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/performance"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestHoistMustCompileFromRangeLoop(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&performance.CompileRegexOutsideLoop{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "regexp"

			func f(inputs []string) {
				for _, s := range inputs {
					re := regexp.MustCompile("\\d+")
					_ = re.MatchString(s)
				}
			}
		`, `
			package main

			import "regexp"

			func f(inputs []string) {
				var compiledRegex0 = regexp.MustCompile("\\d+")
				for _, s := range inputs {
					re := compiledRegex0
					_ = re.MatchString(s)
				}
			}
		`),
	)
}

func TestHoistCompileFromClassicForLoop(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&performance.CompileRegexOutsideLoop{})
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
		`, `
			package main

			import "regexp"

			func f() {
				var compiledRegex0 = regexp.MustCompile("\\d+")
				for i := 0; i < 10; i++ {
					re, _ := compiledRegex0, nil
					_ = re
				}
			}
		`),
	)
}

func TestNoChangeWhenDynamicPattern(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&performance.CompileRegexOutsideLoop{})
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

func TestNoChangeOutsideLoop(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&performance.CompileRegexOutsideLoop{})
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
