/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes-code-quality/recipes/errorhandling"
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

			import "database/sql"

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

			import "database/sql"

			func f(err error) bool {
				return !errors.Is(err, sql.ErrNoRows)
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
