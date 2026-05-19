/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes-code-quality/recipes/style"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestEnsureSqlConnectionClosed(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.EnsureSqlConnectionClosed{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "database/sql"

			func f() {
				db, err := sql.Open("driver", "dsn")
				_ = err
				_ = db
			}
		`, `
			package main

			import "database/sql"

			func f() {
				db, err := sql.Open("driver", "dsn")
				defer db.Close()
				_ = err
				_ = db
			}
		`),
	)
}

func TestEnsureSqlConnectionClosedNoChange(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.EnsureSqlConnectionClosed{})
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

func TestEnsureSqlConnectionClosedAlreadyDeferred(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.EnsureSqlConnectionClosed{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "database/sql"

			func f() {
				db, err := sql.Open("driver", "dsn")
				defer db.Close()
				_ = err
			}
		`),
	)
}
