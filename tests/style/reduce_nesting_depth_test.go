/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes/style"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestReduceNestingDepthGuardClause(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.ReduceNestingDepth{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() {
				err := doSomething()
				if err == nil {
					process()
				}
			}

			func doSomething() error { return nil }
			func process()           {}
		`, `
			package main

			func f() {
				err := doSomething()
				if err != nil {
					return
				}
				process()
			}

			func doSomething() error { return nil }
			func process()           {}
		`),
	)
}

func TestReduceNestingDepthNoChangeNotErrEqualNil(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.ReduceNestingDepth{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() {
				if true {
					x := 1
					_ = x
				}
			}
		`),
	)
}

// Skips a value-returning function.
func TestReduceNestingDepthNoChangeValueReturningFunc(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.ReduceNestingDepth{})
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
		`),
	)
}

// Skips a non-terminal `if err == nil`.
func TestReduceNestingDepthNoChangeNotLastStatement(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.ReduceNestingDepth{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() {
				err := doSomething()
				if err == nil {
					process()
				}
				cleanup()
			}

			func doSomething() error { return nil }
			func process()           {}
			func cleanup()           {}
		`),
	)
}

// Skips an `if err == nil` in a loop body.
func TestReduceNestingDepthNoChangeInsideLoop(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.ReduceNestingDepth{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(xs []int) {
				for _, x := range xs {
					err := check(x)
					if err == nil {
						process(x)
					}
				}
			}

			func check(int) error { return nil }
			func process(int)      {}
		`),
	)
}

func TestReduceNestingDepthNoChangeHasElse(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.ReduceNestingDepth{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() {
				var err error
				if err == nil {
					process()
				} else {
					handleError()
				}
			}

			func process()     {}
			func handleError() {}
		`),
	)
}
