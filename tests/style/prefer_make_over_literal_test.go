/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes/style"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestPreferMakeForEmptyMap(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.PreferMakeForEmptyMap{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() {
				m := map[string]int{}
				_ = m
			}
		`, `
			package main

			func f() {
				m := make(map[string]int)
				_ = m
			}
		`),
	)
}

func TestPreferMakeForEmptyMapNestedMapType(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.PreferMakeForEmptyMap{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() {
				m := map[string]map[int][]byte{}
				_ = m
			}
		`, `
			package main

			func f() {
				m := make(map[string]map[int][]byte)
				_ = m
			}
		`),
	)
}

func TestPreferMakeForEmptyMapNoChangeNonEmptyLiteral(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.PreferMakeForEmptyMap{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() {
				m := map[string]int{"a": 1}
				_ = m
			}
		`),
	)
}

func TestPreferMakeForEmptyMapNoChangeSliceLiteral(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.PreferMakeForEmptyMap{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() {
				s := []string{}
				_ = s
			}
		`),
	)
}

func TestPreferMakeForEmptyMapNoChangeAddressOf(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.PreferMakeForEmptyMap{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() {
				m := &map[string]int{}
				_ = m
			}
		`),
	)
}
