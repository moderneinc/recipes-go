/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes/style"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestReduceErrorCheckNesting(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.ReduceErrorCheckNesting{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() (err error) {
				err = doSomething()
				if err == nil {
					process()
				}
			}

			func doSomething() error { return nil }
			func process()           {}
		`, `
			package main

			func f() (err error) {
				err = doSomething()
				if err != nil {
					return err
				}
				process()
			}

			func doSomething() error { return nil }
			func process()           {}
		`),
	)
}

// Skips a function that does not return a single error.
func TestReduceErrorCheckNestingNoChangeNonErrorReturn(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.ReduceErrorCheckNesting{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func load() int {
				err := doSomething()
				if err == nil {
					return 42
				}
				return 0
			}

			func doSomething() error { return nil }
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
