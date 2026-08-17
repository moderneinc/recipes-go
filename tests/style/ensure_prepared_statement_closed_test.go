/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes/style"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestEnsurePreparedStatementClosed(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.EnsurePreparedStatementClosed{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "database/sql"

			func f(db *sql.DB) {
				stmt, err := db.Prepare("SELECT 1")
				_ = err
				_ = stmt
			}
		`, `
			package main

			import "database/sql"

			func f(db *sql.DB) {
				stmt, err := db.Prepare("SELECT 1")
				defer stmt.Close()
				_ = err
				_ = stmt
			}
		`),
	)
}

func TestEnsurePreparedStatementClosedAfterErrorCheck(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.EnsurePreparedStatementClosed{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "database/sql"

			func f(db *sql.DB) error {
				stmt, err := db.Prepare("SELECT 1")
				if err != nil {
					return err
				}
				_ = stmt
				return nil
			}
		`, `
			package main

			import "database/sql"

			func f(db *sql.DB) error {
				stmt, err := db.Prepare("SELECT 1")
				if err != nil {
					return err
				}
				defer stmt.Close()
				_ = stmt
				return nil
			}
		`),
	)
}

// Only a *sql.Stmt has to be closed; an unrelated `Prepare` must be left alone.
func TestEnsurePreparedStatementClosedNoChangeForeignPrepare(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.EnsurePreparedStatementClosed{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			type builder struct{}

			func (builder) Prepare(q string) (string, error) { return q, nil }

			func f(b builder) {
				s, err := b.Prepare("SELECT 1")
				_, _ = s, err
			}
		`),
	)
}

func TestEnsurePreparedStatementClosedAlreadyDeferred(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.EnsurePreparedStatementClosed{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "database/sql"

			func f(db *sql.DB) {
				stmt, err := db.Prepare("SELECT 1")
				defer stmt.Close()
				_ = err
			}
		`),
	)
}
