/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes/style"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestEnsureTransactionFinalized(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.EnsureTransactionFinalized{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "database/sql"

			func f(db *sql.DB) {
				tx, err := db.Begin()
				_ = err
				_ = tx
			}
		`, `
			package main

			import "database/sql"

			func f(db *sql.DB) {
				tx, err := db.Begin()
				defer tx.Rollback()
				_ = err
				_ = tx
			}
		`),
	)
}

func TestEnsureTransactionFinalizedAfterErrorCheck(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.EnsureTransactionFinalized{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "database/sql"

			func f(db *sql.DB) error {
				tx, err := db.Begin()
				if err != nil {
					return err
				}
				_ = tx
				return nil
			}
		`, `
			package main

			import "database/sql"

			func f(db *sql.DB) error {
				tx, err := db.Begin()
				if err != nil {
					return err
				}
				defer tx.Rollback()
				_ = tx
				return nil
			}
		`),
	)
}

// Only a *sql.Tx has to be rolled back; an unrelated `Begin` must be left alone.
func TestEnsureTransactionFinalizedNoChangeForeignBegin(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.EnsureTransactionFinalized{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			type span struct{}

			func (span) Begin() (string, error) { return "", nil }

			func f(s span) {
				id, err := s.Begin()
				_, _ = id, err
			}
		`),
	)
}

func TestEnsureTransactionFinalizedNoChange(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.EnsureTransactionFinalized{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "database/sql"

			func f(db *sql.DB) {
				db.Close()
			}
		`),
	)
}

func TestEnsureTransactionFinalizedAlreadyDeferred(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.EnsureTransactionFinalized{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "database/sql"

			func f(db *sql.DB) {
				tx, err := db.Begin()
				defer tx.Rollback()
				_ = err
			}
		`),
	)
}
