/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes-code-quality/recipes/style"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestReduceErrorCheckNesting(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.ReduceErrorCheckNesting{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() error {
				err := doSomething()
				if err == nil {
					process()
				}
				return nil
			}

			func doSomething() error { return nil }
			func process()           {}
		`, `
			package main

			func f() error {
				err := doSomething()
				if err != nil {
					return err
				}
				process()
				return nil
			}

			func doSomething() error { return nil }
			func process()           {}
		`),
	)
}

func TestReduceErrorCheckNestingNoChangeErrNotNil(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.ReduceErrorCheckNesting{})
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

func TestReduceErrorCheckNestingNoChangeHasElse(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.ReduceErrorCheckNesting{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() error {
				var err error
				if err == nil {
					process()
				} else {
					handleError()
				}
				return nil
			}

			func doSomething() error { return nil }
			func process()           {}
			func handleError()       {}
		`),
	)
}
