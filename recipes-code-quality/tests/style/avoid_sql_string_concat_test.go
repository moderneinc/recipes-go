/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes-code-quality/recipes/style"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestAvoidSqlStringConcat(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.AvoidSqlStringConcat{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(table string) string {
				return "SELECT * FROM " + table
			}
		`, `
			package main

			func f(table string) string {
				return/*~~(possible SQL injection via string concatenation)~~>*/ "SELECT * FROM " + table
			}
		`),
	)
}

func TestAvoidSqlStringConcatNoChange(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.AvoidSqlStringConcat{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(name string) string {
				return "hello " + name
			}
		`),
	)
}
