/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package testify_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes/migration/testify"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

// `if len(x) != n` asserts the length equals n.
func TestAdoptRequireLenFromNotEqual(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&testify.AdoptTestifyRequireLen{})
	spec.RewriteRun(t,
		test.Golang(`
			package sample

			import "testing"

			func build() []int { return nil }

			func TestThing(t *testing.T) {
				s := build()
				if len(s) != 3 {
					t.Fatalf("wrong length")
				}
			}
		`, `
			package sample

			import (
				"testing"
				"github.com/stretchr/testify/require"
			)

			func build() []int { return nil }

			func TestThing(t *testing.T) {
				s := build()
				require.Len(t, s, 3, "wrong length")
			}
		`),
	)
}

// The count may come first; the len() side is the collection.
func TestAdoptAssertLenCountFirst(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&testify.AdoptTestifyAssertLen{})
	spec.RewriteRun(t,
		test.Golang(`
			package sample

			import "testing"

			func build() []int { return nil }

			func TestThing(t *testing.T) {
				s := build()
				if 0 != len(s) {
					t.Errorf("not empty")
				}
			}
		`, `
			package sample

			import (
				"testing"
				"github.com/stretchr/testify/assert"
			)

			func build() []int { return nil }

			func TestThing(t *testing.T) {
				s := build()
				assert.Len(t, s, 0, "not empty")
			}
		`),
	)
}

// `len(x) == n` asserts a non-length; Len cannot express it, so no change.
func TestAdoptRequireLenNoChangeOnEqual(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&testify.AdoptTestifyRequireLen{})
	spec.RewriteRun(t,
		test.Golang(`
			package sample

			import "testing"

			func build() []int { return nil }

			func TestThing(t *testing.T) {
				s := build()
				if len(s) == 0 {
					t.Fatal("empty")
				}
			}
		`),
	)
}
