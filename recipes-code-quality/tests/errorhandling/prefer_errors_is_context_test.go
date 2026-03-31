/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/errorhandling"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestPreferErrorsIsContextCanceledEqual(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.PreferErrorsIsContext{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "context"

			func f(err error) bool {
				return err == context.Canceled
			}
		`, `
			package main

			import "context"

			func f(err error) bool {
				return errors.Is(err, context.Canceled)
			}
		`),
	)
}

func TestPreferErrorsIsContextCanceledNotEqual(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.PreferErrorsIsContext{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "context"

			func f(err error) bool {
				return err != context.Canceled
			}
		`, `
			package main

			import "context"

			func f(err error) bool {
				return !errors.Is(err, context.Canceled)
			}
		`),
	)
}

func TestPreferErrorsIsContextDeadlineEqual(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.PreferErrorsIsContext{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "context"

			func f(err error) bool {
				return err == context.DeadlineExceeded
			}
		`, `
			package main

			import "context"

			func f(err error) bool {
				return errors.Is(err, context.DeadlineExceeded)
			}
		`),
	)
}

func TestPreferErrorsIsContextDeadlineNotEqual(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.PreferErrorsIsContext{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "context"

			func f(err error) bool {
				return err != context.DeadlineExceeded
			}
		`, `
			package main

			import "context"

			func f(err error) bool {
				return !errors.Is(err, context.DeadlineExceeded)
			}
		`),
	)
}

func TestPreferErrorsIsContextNoChangeNilCheck(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.PreferErrorsIsContext{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(err error) bool {
				return err == nil
			}
		`),
	)
}
