/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes/errorhandling"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestPreferErrorsIsForFieldAccessEqual(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.PreferErrorsIsForFieldAccess{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "database/sql"

			func f(err error) bool {
				return err == sql.ErrNoRows
			}
		`, `
			package main

			import (
				"database/sql"
				"errors"
			)

			func f(err error) bool {
				return errors.Is(err, sql.ErrNoRows)
			}
		`),
	)
}

func TestPreferErrorsIsForFieldAccessNotEqual(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.PreferErrorsIsForFieldAccess{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "database/sql"

			func f(err error) bool {
				return err != sql.ErrNoRows
			}
		`, `
			package main

			import (
				"database/sql"
				"errors"
			)

			func f(err error) bool {
				return !errors.Is(err, sql.ErrNoRows)
			}
		`),
	)
}

func TestPreferErrorsIsForFieldAccessEOFAddsErrorsImport(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.PreferErrorsIsForFieldAccess{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "io"

			func f(err error) bool {
				return err == io.EOF
			}
		`, `
			package main

			import (
				"io"
				"errors"
			)

			func f(err error) bool {
				return errors.Is(err, io.EOF)
			}
		`),
	)
}

func TestPreferErrorsIsForFieldAccessNoChangeNil(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.PreferErrorsIsForFieldAccess{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(err error) bool {
				return err == nil
			}
		`),
	)
}

func TestPreferErrorsIsForFieldAccessNoChangeNonSentinel(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.PreferErrorsIsForFieldAccess{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(x, y int) bool {
				return x == y
			}
		`),
	)
}

// Skips a non-error field comparison that only matched by an Err* field name.
func TestPreferErrorsIsForFieldAccessNoChangeNonError(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.PreferErrorsIsForFieldAccess{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			type C struct{ ErrThreshold int }

			func f(c C, n int) bool {
				return n == c.ErrThreshold
			}
		`),
	)
}
