/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes/errorhandling"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestHandleCheckedErrorSimple(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.HandleCheckedError{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() error {
				err := doSomething()
				if err != nil {
				}
				return nil
			}

			func doSomething() error { return nil }
		`, `
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

func TestHandleCheckedErrorNoChangeVoidFunc(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.HandleCheckedError{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() {
				err := doSomething()
				if err != nil {
				}
				_ = err
			}

			func doSomething() error { return nil }
		`),
	)
}

func TestHandleCheckedErrorNoChangeHandled(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.HandleCheckedError{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "log"

			func f() {
				err := doSomething()
				if err != nil {
					log.Fatal(err)
				}
			}

			func doSomething() error { return nil }
		`),
	)
}

func TestHandleCheckedErrorNoChangeNilCheck(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.HandleCheckedError{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() {
				x := 1
				if x != 0 {
				}
			}
		`),
	)
}
