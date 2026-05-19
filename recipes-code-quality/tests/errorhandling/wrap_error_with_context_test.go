/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes-code-quality/recipes/errorhandling"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestWrapErrorWithContextBareReturn(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.WrapErrorWithContext{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() error {
				err := doSomething()
				return err
			}

			func doSomething() error { return nil }
		`, `
			package main

			func f() error {
				err := doSomething()
				return fmt.Errorf("f: %w", err)
			}

			func doSomething() error { return nil }
		`),
	)
}

func TestWrapErrorWithContextNoChangeWrapped(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.WrapErrorWithContext{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "fmt"

			func f() error {
				err := doSomething()
				return fmt.Errorf("context: %w", err)
			}

			func doSomething() error { return nil }
		`),
	)
}

func TestWrapErrorWithContextNoChangeNil(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.WrapErrorWithContext{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() error {
				return nil
			}
		`),
	)
}

func TestWrapErrorWithContextNoChangeMultiReturn(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.WrapErrorWithContext{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() (int, error) {
				return 0, err
			}

			var err error
		`),
	)
}
