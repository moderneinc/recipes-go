/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/errorhandling"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestFindErrorStringComparison(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.FindErrorStringComparison{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(err error) bool {
				return err.Error() == "not found"
			}
		`),
	)
}

func TestFindErrorStringComparisonNoChangeSentinel(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.FindErrorStringComparison{})
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
