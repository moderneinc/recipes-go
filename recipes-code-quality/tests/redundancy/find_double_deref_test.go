/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package redundancy_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/redundancy"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestFindDoubleDeref(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.FindDoubleDeref{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() {
				x := 42
				y := *&x
				_ = y
			}
		`),
	)
}

func TestFindDoubleDerefNoChangePlainDeref(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.FindDoubleDeref{})
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

func TestFindDoubleDerefNoChangePlainAddressOf(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.FindDoubleDeref{})
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
