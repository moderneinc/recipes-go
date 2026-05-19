/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes-code-quality/recipes/style"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestReduceNestingDepthGuardClause(t *testing.T) {
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
		`, `
			package main

			func f() error {
				err := doSomething()
				if err != nil {
					return
				}
				process()
				return nil
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
