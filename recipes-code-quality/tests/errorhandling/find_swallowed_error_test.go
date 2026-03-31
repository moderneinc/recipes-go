/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/errorhandling"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestFindSwallowedError(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.FindSwallowedError{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() {
				err := doSomething()
				if err != nil {
					return
				}
			}

			func doSomething() error { return nil }
		`),
	)
}

func TestFindSwallowedErrorNoChangeReturnsErr(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.FindSwallowedError{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() error {
				err := doSomething()
				if err != nil {
					return err
				}
				return nil
			}

			func doSomething() error { return nil }
		`),
	)
}
