/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/style"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestEnsureSqlRowsClosed(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.EnsureSqlRowsClosed{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(db interface{ Query(string, ...any) (interface{ Close() }, error) }) {
				rows, err := db.Query("SELECT 1")
				_ = err
				_ = rows
			}
		`, `
			package main

			func f(db interface{ Query(string, ...any) (interface{ Close() }, error) }) {
				rows, err := db.Query("SELECT 1")
				defer rows.Close()
				_ = err
				_ = rows
			}
		`),
	)
}

func TestEnsureSqlRowsClosedNoChange(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.EnsureSqlRowsClosed{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(db interface{ QueryRow(string, ...any) }) {
				db.QueryRow("SELECT 1")
			}
		`),
	)
}

func TestEnsureSqlRowsClosedAlreadyDeferred(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.EnsureSqlRowsClosed{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(db interface{ Query(string, ...any) (interface{ Close() }, error) }) {
				rows, err := db.Query("SELECT 1")
				defer rows.Close()
				_ = err
			}
		`),
	)
}
