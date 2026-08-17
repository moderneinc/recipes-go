/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes/style"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestEnsureSqlRowsClosed(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.EnsureSqlRowsClosed{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "database/sql"

			func f(db *sql.DB) {
				rows, err := db.Query("SELECT 1")
				_ = err
				_ = rows
			}
		`, `
			package main

			import "database/sql"

			func f(db *sql.DB) {
				rows, err := db.Query("SELECT 1")
				defer rows.Close()
				_ = err
				_ = rows
			}
		`),
	)
}

func TestEnsureSqlRowsClosedQueryContext(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.EnsureSqlRowsClosed{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import (
				"context"
				"database/sql"
			)

			func f(ctx context.Context, db *sql.DB) {
				rows, err := db.QueryContext(ctx, "SELECT 1")
				_ = err
				_ = rows
			}
		`, `
			package main

			import (
				"context"
				"database/sql"
			)

			func f(ctx context.Context, db *sql.DB) {
				rows, err := db.QueryContext(ctx, "SELECT 1")
				defer rows.Close()
				_ = err
				_ = rows
			}
		`),
	)
}

func TestEnsureSqlRowsClosedAfterErrorCheck(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.EnsureSqlRowsClosed{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "database/sql"

			func f(db *sql.DB) error {
				rows, err := db.Query("SELECT 1")
				if err != nil {
					return err
				}
				_ = rows
				return nil
			}
		`, `
			package main

			import "database/sql"

			func f(db *sql.DB) error {
				rows, err := db.Query("SELECT 1")
				if err != nil {
					return err
				}
				defer rows.Close()
				_ = rows
				return nil
			}
		`),
	)
}

// url.Values has no Close method, so a `Query` that is not a SQL query must be
// left alone.
func TestEnsureSqlRowsClosedNoChangeUrlQuery(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.EnsureSqlRowsClosed{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "net/http"

			func handler(r *http.Request) {
				q := r.URL.Query()
				_ = q
			}
		`),
	)
}

func TestEnsureSqlRowsClosedNoChangeQueryRow(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.EnsureSqlRowsClosed{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "database/sql"

			func f(db *sql.DB) {
				row := db.QueryRow("SELECT 1")
				_ = row
			}
		`),
	)
}

func TestEnsureSqlRowsClosedAlreadyDeferred(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.EnsureSqlRowsClosed{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "database/sql"

			func f(db *sql.DB) {
				rows, err := db.Query("SELECT 1")
				defer rows.Close()
				_ = err
			}
		`),
	)
}
