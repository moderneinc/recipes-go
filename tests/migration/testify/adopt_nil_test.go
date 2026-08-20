/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package testify_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes/migration/testify"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

// `if p != nil` asserts the pointer is nil.
func TestAdoptRequireNilFromNotEqual(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&testify.AdoptTestifyRequireNil{})
	spec.RewriteRun(t,
		test.Golang(`
			package sample

			import "testing"

			func get() *int { return nil }

			func TestThing(t *testing.T) {
				p := get()
				if p != nil {
					t.Fatal("expected nil")
				}
			}
		`, `
			package sample

			import (
				"testing"

				"github.com/stretchr/testify/require"
			)

			func get() *int { return nil }

			func TestThing(t *testing.T) {
				p := get()
				require.Nil(t, p, "expected nil")
			}
		`),
	)
}

// `if p == nil` asserts the pointer is not nil.
func TestAdoptAssertNotNilFromEqual(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&testify.AdoptTestifyAssertNil{})
	spec.RewriteRun(t,
		test.Golang(`
			package sample

			import "testing"

			func get() *int { return nil }

			func TestThing(t *testing.T) {
				p := get()
				if p == nil {
					t.Error("expected non-nil")
				}
			}
		`, `
			package sample

			import (
				"testing"

				"github.com/stretchr/testify/assert"
			)

			func get() *int { return nil }

			func TestThing(t *testing.T) {
				p := get()
				assert.NotNil(t, p, "expected non-nil")
			}
		`),
	)
}

// An error operand belongs to the NoError recipe, not Nil.
func TestAdoptRequireNilNoChangeOnError(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&testify.AdoptTestifyRequireNil{})
	spec.RewriteRun(t,
		test.Golang(`
			package sample

			import "testing"

			func do() error { return nil }

			func TestThing(t *testing.T) {
				err := do()
				if err != nil {
					t.Fatal(err)
				}
			}
		`),
	)
}
