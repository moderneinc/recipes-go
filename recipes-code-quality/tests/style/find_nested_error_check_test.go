/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/style"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestFindDeeplyNestedErrorCheck(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindDeeplyNestedErrorCheck{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() error {
				if true {
					if true {
						err := doSomething()
						if err != nil {
							return err
						}
					}
				}
				return nil
			}

			func doSomething() error { return nil }
		`),
	)
}

func TestFindDeeplyNestedErrorCheckNoChangeShallow(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindDeeplyNestedErrorCheck{})
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
