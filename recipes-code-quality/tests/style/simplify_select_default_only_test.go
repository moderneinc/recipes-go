/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes-code-quality/recipes/style"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestSimplifySelectDefaultOnly(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.SimplifySelectDefaultOnly{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() {
				select {
				default:
					x()
				}
			}
		`, `
			package main

			func f() {
				x()
			}
		`),
	)
}

func TestSimplifySelectDefaultOnlyNoChangeWithCase(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.SimplifySelectDefaultOnly{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(ch chan int) {
				select {
				case <-ch:
					x()
				}
			}
		`),
	)
}
