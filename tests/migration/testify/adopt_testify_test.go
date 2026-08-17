/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package testify_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes/migration/testify"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

// The aggregate transforms several guard shapes and adds the go.mod dependency in
// one run: an error guard, a length check, and an equality check (the length
// check must become Len, not Equal).
func TestAdoptTestifyEndToEnd(t *testing.T) {
	// given a module with hand-written guards and no testify dependency
	spec := test.NewRecipeSpec().WithRecipe(&testify.AdoptTestify{})

	// when / then each guard is converted and go.mod gains the require
	spec.RewriteRun(t,
		test.GoProject("app",
			test.GoMod(`
				module example.com/app

				go 1.22

				require (
					github.com/foo/bar v1.2.3
				)
			`, `
				module example.com/app

				go 1.22

				require (
					github.com/foo/bar v1.2.3
					github.com/stretchr/testify v1.10.0
				)
			`),
			test.Golang(`
				package app

				import "testing"

				func do() error { return nil }
				func compute() int { return 0 }
				func build() []int { return nil }

				func TestThing(t *testing.T) {
					err := do()
					if err != nil {
						t.Fatal(err)
					}
					s := build()
					if len(s) != 3 {
						t.Fatal("len")
					}
					n := compute()
					if n != 42 {
						t.Fatalf("wrong")
					}
				}
			`, `
				package app

				import (
					"testing"
					"github.com/stretchr/testify/require"
				)

				func do() error { return nil }
				func compute() int { return 0 }
				func build() []int { return nil }

				func TestThing(t *testing.T) {
					err := do()
					require.NoError(t, err)
					s := build()
					require.Len(t, s, 3, "len")
					n := compute()
					require.Equal(t, 42, n, "wrong")
				}
			`),
		),
	)
}
