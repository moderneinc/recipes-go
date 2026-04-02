/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package redundancy_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/redundancy"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestRemoveDoubleDeref(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.RemoveDoubleDeref{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() {
				x := 42
				y := *&x
				_ = y
			}
		`, `
			package main

			func f() {
				x := 42
				y := x
				_ = y
			}
		`),
	)
}

func TestRemoveDoubleDerefNoChangePlainDeref(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.RemoveDoubleDeref{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() {
				x := 42
				p := &x
				y := *p
				_ = y
			}
		`),
	)
}

func TestRemoveDoubleDerefNoChangePlainAddressOf(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.RemoveDoubleDeref{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() {
				x := 42
				p := &x
				_ = p
			}
		`),
	)
}
