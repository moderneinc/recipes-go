/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/style"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestFindSelectDefaultOnly(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindSelectDefaultOnly{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() {
				select {
				default:
					x()
				}
			}
		`),
	)
}

func TestFindSelectDefaultOnlyNoChangeWithCase(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindSelectDefaultOnly{})
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
