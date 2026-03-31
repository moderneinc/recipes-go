/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/errorhandling"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestPreferErrorsIsSimple(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.PreferErrorsIsOverEquality{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "errors"

			var ErrNotFound = errors.New("not found")

			func f(err error) bool {
				return err == ErrNotFound
			}
		`, `
			package main

			import "errors"

			var ErrNotFound = errors.New("not found")

			func f(err error) bool {
				return errors.Is(err, ErrNotFound)
			}
		`),
	)
}

func TestPreferErrorsIsNotEqual(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.PreferErrorsIsOverEquality{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "errors"

			var ErrNotFound = errors.New("not found")

			func f(err error) bool {
				return err != ErrNotFound
			}
		`, `
			package main

			import "errors"

			var ErrNotFound = errors.New("not found")

			func f(err error) bool {
				return !errors.Is(err, ErrNotFound)
			}
		`),
	)
}

func TestPreferErrorsIsNoChangeNilCheck(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.PreferErrorsIsOverEquality{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(err error) bool {
				return err == nil
			}
		`),
	)
}

func TestPreferErrorsIsNoChangeNonError(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.PreferErrorsIsOverEquality{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(x, y int) bool {
				return x == y
			}
		`),
	)
}
