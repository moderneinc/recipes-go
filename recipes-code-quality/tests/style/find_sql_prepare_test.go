/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/style"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestFindSqlPrepare(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindSqlPrepare{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(db interface{ Prepare(string) }) {
				db.Prepare("SELECT ?")
			}
		`),
	)
}

func TestFindSqlPrepareNoChange(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindSqlPrepare{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(db interface{ Query(string, ...any) }) {
				db.Query("SELECT 1")
			}
		`),
	)
}
