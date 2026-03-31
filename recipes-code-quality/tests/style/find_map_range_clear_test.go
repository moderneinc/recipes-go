/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/style"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestFindMapRangeClear(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindMapRangeClear{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() {
				m := map[string]int{"a": 1}
				for k := range m {
					delete(m, k)
				}
			}
		`),
	)
}

func TestFindMapRangeClearNoChangeRegularRange(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindMapRangeClear{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() {
				m := map[string]int{"a": 1}
				for k, v := range m {
					println(k, v)
				}
			}
		`),
	)
}
