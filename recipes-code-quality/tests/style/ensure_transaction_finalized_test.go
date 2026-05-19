/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes-code-quality/recipes/style"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestEnsureTransactionFinalized(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.EnsureTransactionFinalized{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(db interface{ Begin() (interface{ Rollback() }, error) }) {
				tx, err := db.Begin()
				_ = err
				_ = tx
			}
		`, `
			package main

			func f(db interface{ Begin() (interface{ Rollback() }, error) }) {
				tx, err := db.Begin()
				defer tx.Rollback()
				_ = err
				_ = tx
			}
		`),
	)
}

func TestEnsureTransactionFinalizedNoChange(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.EnsureTransactionFinalized{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(db interface{ Close() }) {
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

			func f(db interface{ Begin() (interface{ Rollback() }, error) }) {
				tx, err := db.Begin()
				defer tx.Rollback()
				_ = err
			}
		`),
	)
}
