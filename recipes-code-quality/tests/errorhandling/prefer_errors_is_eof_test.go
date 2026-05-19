/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes-code-quality/recipes/errorhandling"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestPreferErrorsIsEOFEqual(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.PreferErrorsIsEOF{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "io"

			func f(err error) bool {
				return err == io.EOF
			}
		`, `
			package main

			import "io"

			func f(err error) bool {
				return errors.Is(err, io.EOF)
			}
		`),
	)
}

func TestPreferErrorsIsEOFNotEqual(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.PreferErrorsIsEOF{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "io"

			func f(err error) bool {
				return err != io.EOF
			}
		`, `
			package main

			import "io"

			func f(err error) bool {
				return !errors.Is(err, io.EOF)
			}
		`),
	)
}

func TestPreferErrorsIsEOFNoChangeNil(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.PreferErrorsIsEOF{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(err error) bool {
				return err == nil
			}
		`),
	)
}
