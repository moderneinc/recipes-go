/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/style"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestEnsurePreparedStatementClosed(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.EnsurePreparedStatementClosed{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(db interface{ Prepare(string) (interface{ Close() }, error) }) {
				stmt, err := db.Prepare("SELECT ?")
				_ = err
				_ = stmt
			}
		`, `
			package main

			func f(db interface{ Prepare(string) (interface{ Close() }, error) }) {
				stmt, err := db.Prepare("SELECT ?")
				defer stmt.Close()
				_ = err
				_ = stmt
			}
		`),
	)
}

func TestEnsurePreparedStatementClosedNoChange(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.EnsurePreparedStatementClosed{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(db interface{ Query(string, ...any) }) {
				db.Query("SELECT 1")
			}
		`),
	)
}

func TestEnsurePreparedStatementClosedAlreadyDeferred(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.EnsurePreparedStatementClosed{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(db interface{ Prepare(string) (interface{ Close() }, error) }) {
				stmt, err := db.Prepare("SELECT ?")
				defer stmt.Close()
				_ = err
			}
		`),
	)
}
