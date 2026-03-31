/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/style"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestFindSqlQueryConcat(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindSqlQueryConcat{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(db interface{ Query(string, ...any) }, table string) {
				db.Query("SELECT * FROM " + table)
			}
		`),
	)
}

func TestFindSqlQueryConcatNoChange(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindSqlQueryConcat{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(db interface{ Query(string, ...any) }) {
				db.Query("SELECT * FROM users")
			}
		`),
	)
}
