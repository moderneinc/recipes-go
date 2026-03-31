/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/style"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestFindSqlBegin(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindSqlBegin{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(db interface{ Begin() }) {
				db.Begin()
			}
		`),
	)
}

func TestFindSqlBeginNoChange(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindSqlBegin{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(db interface{ Close() }) {
				db.Close()
			}
		`),
	)
}
