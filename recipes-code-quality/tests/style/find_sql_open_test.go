/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/style"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestFindSqlOpen(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindSqlOpen{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "database/sql"

			func f() {
				sql.Open("driver", "dsn")
			}
		`),
	)
}

func TestFindSqlOpenNoChange(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindSqlOpen{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "database/sql"

			func f() {
				sql.Register("driver", nil)
			}
		`),
	)
}
