/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/simplification"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestSimplifyErrorsIsNil(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.SimplifyErrorsIsNil{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "errors"

			func f(err error) bool {
				return errors.Is(err, nil)
			}
		`, `
			package main

			import "errors"

			func f(err error) bool {
				return err == nil
			}
		`),
	)
}
