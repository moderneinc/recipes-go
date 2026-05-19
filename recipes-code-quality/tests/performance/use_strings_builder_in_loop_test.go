/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package performance_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes-code-quality/recipes/performance"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestUseBuilderInRangeLoop(t *testing.T) {
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
		`, `
			package main

			import "strings"

			func f(items []string) string {
				s := ""
				var builder strings.Builder
				for _, item := range items {
					builder.WriteString(item)
				}
				s = builder.String()
				return s
			}
		`),
	)
}

func TestUseBuilderInClassicForLoop(t *testing.T) {
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
		`, `
			package main

			import "strings"

			func f() string {
				s := ""
				var builder strings.Builder
				for i := 0; i < 10; i++ {
					builder.WriteString("x")
				}
				s = builder.String()
				return s
			}
		`),
	)
}

func TestStringConcatNoChangeOutsideLoop(t *testing.T) {
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
