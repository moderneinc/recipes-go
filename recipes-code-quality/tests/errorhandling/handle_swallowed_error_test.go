/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes-code-quality/recipes/errorhandling"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestHandleSwallowedError(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.HandleSwallowedError{})
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
		`, `
			package main

			func f() {
				err := doSomething()
				if err != nil {
					return err
				}
			}

			func doSomething() error { return nil }
		`),
	)
}

func TestHandleSwallowedErrorNoChangeReturnsErr(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.HandleSwallowedError{})
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
