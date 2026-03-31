/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/errorhandling"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestPreferErrorsIsHttpServerClosedEqual(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.PreferErrorsIsHttpServerClosed{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "net/http"

			func f(err error) bool {
				return err == http.ErrServerClosed
			}
		`, `
			package main

			import "net/http"

			func f(err error) bool {
				return errors.Is(err, http.ErrServerClosed)
			}
		`),
	)
}

func TestPreferErrorsIsHttpServerClosedNotEqual(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.PreferErrorsIsHttpServerClosed{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "net/http"

			func f(err error) bool {
				return err != http.ErrServerClosed
			}
		`, `
			package main

			import "net/http"

			func f(err error) bool {
				return !errors.Is(err, http.ErrServerClosed)
			}
		`),
	)
}

func TestPreferErrorsIsHttpServerClosedNoChangeNil(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.PreferErrorsIsHttpServerClosed{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(err error) bool {
				return err == nil
			}
		`),
	)
}
