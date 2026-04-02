/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package redundancy_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/redundancy"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestRemoveRedundantReturnSimple(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.RemoveRedundantReturn{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func hello() {
				println("hello")
				return
			}
		`, `
			package main

			func hello() {
				println("hello")
			}
		`),
	)
}

func TestRemoveRedundantReturnOnlyReturn(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.RemoveRedundantReturn{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func noop() {
				return
			}
		`, `
			package main

			func noop() {
			}
		`),
	)
}

func TestRemoveRedundantReturnNoChangeWithReturnValue(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.RemoveRedundantReturn{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func add(a, b int) int {
				return a + b
			}
		`),
	)
}

func TestRemoveRedundantReturnNoChangeNoReturn(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.RemoveRedundantReturn{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func hello() {
				println("hello")
			}
		`),
	)
}
