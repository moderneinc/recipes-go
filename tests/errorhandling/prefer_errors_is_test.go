/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes/errorhandling"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
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

func TestPreferErrorsIsEOFAddsErrorsImport(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.PreferErrorsIsOverEquality{})
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
