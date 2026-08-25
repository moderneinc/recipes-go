/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package performance_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes/performance"
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

// Skips a numeric accumulator.
func TestStringConcatNoChangeNumeric(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&performance.UseStringsBuilderInLoop{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func total(xs []int) int {
				s := 0
				for _, x := range xs {
					s += x
				}
				return s
			}
		`),
	)
}

func TestUseBuilderWithGroupedImports(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&performance.UseStringsBuilderInLoop{})
	spec.RewriteRun(t,
		test.Golang(`
			package sample

			import (
				"fmt"

				"github.com/stretchr/testify/assert"
			)

			func f(items []string) string {
				s := ""
				for _, item := range items {
					s += item
				}
				fmt.Println(assert.ObjectsAreEqual(s, "x"))
				return s
			}
		`, `
			package sample

			import (
				"fmt"
				"strings"

				"github.com/stretchr/testify/assert"
			)

			func f(items []string) string {
				s := ""
				var builder strings.Builder
				for _, item := range items {
					builder.WriteString(item)
				}
				s = builder.String()
				fmt.Println(assert.ObjectsAreEqual(s, "x"))
				return s
			}
		`),
	)
}

func TestUseBuilderWithStringsAlreadyImported(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&performance.UseStringsBuilderInLoop{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "strings"

			func f(items []string) string {
				s := ""
				for _, item := range items {
					s += strings.TrimSpace(item)
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
					builder.WriteString(strings.TrimSpace(item))
				}
				s = builder.String()
				return s
			}
		`),
	)
}

func TestUseBuilderKeepsCommentOnLoop(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&performance.UseStringsBuilderInLoop{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(items []string) string {
				s := ""
				// Concatenate every item.
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
				// Concatenate every item.
				for _, item := range items {
					builder.WriteString(item)
				}
				s = builder.String()
				return s
			}
		`),
	)
}
