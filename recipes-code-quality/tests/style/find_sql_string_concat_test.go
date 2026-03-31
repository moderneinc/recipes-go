/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/style"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestFindSQLStringConcat(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindSQLStringConcat{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(table string) string {
				return "SELECT * FROM " + table
			}
		`),
	)
}

func TestFindSQLStringConcatNoChange(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindSQLStringConcat{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(name string) string {
				return "hello " + name
			}
		`),
	)
}
