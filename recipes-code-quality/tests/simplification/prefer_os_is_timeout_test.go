/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/simplification"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestPreferOsIsTimeout(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.PreferOsIsTimeout{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "os"

			func f(err error) bool {
				return os.IsTimeout(err)
			}
		`, `
			package main

			import "os"

			func f(err error) bool {
				return errors.Is(err, os.ErrDeadlineExceeded)
			}
		`),
	)
}
