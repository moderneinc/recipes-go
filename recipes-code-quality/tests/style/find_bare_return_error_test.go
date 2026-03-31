/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/style"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestFindBareReturnNilError(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindBareReturnNilError{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() (int, error) {
				err := doSomething()
				if err != nil {
					return nil, err
				}
				return nil, nil
			}

			func doSomething() error { return nil }
		`),
	)
}

func TestFindBareReturnNilErrorNoChangeWrapped(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindBareReturnNilError{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "fmt"

			func f() (int, error) {
				err := doSomething()
				if err != nil {
					return nil, fmt.Errorf("failed: %w", err)
				}
				return nil, nil
			}

			func doSomething() error { return nil }
		`),
	)
}
