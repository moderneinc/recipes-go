/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package redundancy_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/redundancy"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestRemoveUnconditionalValueOverwriteMap(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.RemoveUnconditionalValueOverwrite{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() {
				m := map[string]int{}
				m["key"] = 1
				m["key"] = 2
			}
		`, `
			package main

			func f() {
				m := map[string]int{}
				m["key"] = 2
			}
		`),
	)
}

func TestRemoveUnconditionalValueOverwriteDifferentKeys(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.RemoveUnconditionalValueOverwrite{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() {
				m := map[string]int{}
				m["a"] = 1
				m["b"] = 2
			}
		`),
	)
}

func TestRemoveUnconditionalValueOverwriteDifferentReceivers(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.RemoveUnconditionalValueOverwrite{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() {
				m := map[string]int{}
				n := map[string]int{}
				m["key"] = 1
				n["key"] = 2
			}
		`),
	)
}

func TestRemoveUnconditionalValueOverwriteNonConsecutive(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.RemoveUnconditionalValueOverwrite{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() {
				m := map[string]int{}
				m["key"] = 1
				println("between")
				m["key"] = 2
			}
		`),
	)
}

func TestRemoveUnconditionalValueOverwriteSlice(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.RemoveUnconditionalValueOverwrite{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(s []int) {
				s[0] = 1
				s[0] = 2
			}
		`, `
			package main

			func f(s []int) {
				s[0] = 2
			}
		`),
	)
}
