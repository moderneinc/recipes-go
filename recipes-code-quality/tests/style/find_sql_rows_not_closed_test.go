/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/style"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestFindSqlQuery(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindSqlQuery{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(db interface{ Query(string, ...any) }) {
				db.Query("SELECT 1")
			}
		`),
	)
}

func TestFindSqlQueryNoChange(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindSqlQuery{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(db interface{ QueryRow(string, ...any) }) {
				db.QueryRow("SELECT 1")
			}
		`),
	)
}
