/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes-code-quality/recipes/errorhandling"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestPreferErrorsIsOsInvalidEqual(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.PreferErrorsIsOsInvalid{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "os"

			func f(err error) bool {
				return err == os.ErrInvalid
			}
		`, `
			package main

			import "os"

			func f(err error) bool {
				return errors.Is(err, os.ErrInvalid)
			}
		`),
	)
}

func TestPreferErrorsIsOsInvalidNoChangeNil(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.PreferErrorsIsOsInvalid{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(err error) bool {
				return err == nil
			}
		`),
	)
}
