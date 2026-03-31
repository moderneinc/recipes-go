/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/errorhandling"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestFindErrorsNew(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.FindErrorsNew{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "errors"

			func f() error {
				return errors.New("fail")
			}
		`),
	)
}

func TestFindErrorsNewNoChangeFmtErrorf(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.FindErrorsNew{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "fmt"

			func f() error {
				return fmt.Errorf("fail")
			}
		`),
	)
}
