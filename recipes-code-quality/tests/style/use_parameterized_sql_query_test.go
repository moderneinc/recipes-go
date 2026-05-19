/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes-code-quality/recipes/style"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestUseParameterizedSqlQuery(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.UseParameterizedSqlQuery{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(db interface{ Query(string, ...any) }, table string) {
				db.Query("SELECT * FROM " + table)
			}
		`),
	)
}

func TestUseParameterizedSqlQueryNoChange(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.UseParameterizedSqlQuery{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(db interface{ Query(string, ...any) }) {
				db.Query("SELECT * FROM users")
			}
		`),
	)
}
