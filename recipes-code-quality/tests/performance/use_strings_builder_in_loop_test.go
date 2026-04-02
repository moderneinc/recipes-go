/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package performance_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/performance"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestFindStringConcatInForLoop(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&performance.UseStringsBuilderInLoop{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(items []string) string {
				s := ""
				for _, item := range items {
					s += item
				}
				return s
			}
		`),
	)
}

func TestFindStringConcatInClassicForLoop(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&performance.UseStringsBuilderInLoop{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() string {
				s := ""
				for i := 0; i < 10; i++ {
					s += "x"
				}
				return s
			}
		`),
	)
}

func TestFindStringConcatNoChangeOutsideLoop(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&performance.UseStringsBuilderInLoop{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() string {
				s := ""
				s += "hello"
				return s
			}
		`),
	)
}
