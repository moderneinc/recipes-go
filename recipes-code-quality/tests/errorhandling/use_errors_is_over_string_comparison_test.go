/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/errorhandling"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestUseErrorsIsOverStringComparison(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.UseErrorsIsOverStringComparison{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(err error) bool {
				return err.Error() == "not found"
			}
		`),
	)
}

func TestUseErrorsIsOverStringComparisonNoChangeSentinel(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.UseErrorsIsOverStringComparison{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "errors"

			var ErrNotFound = errors.New("not found")

			func f(err error) bool {
				return err == ErrNotFound
			}
		`),
	)
}
