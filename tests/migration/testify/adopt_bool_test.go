/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package testify_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes/migration/testify"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

// `if !cond` asserts the condition is true.
func TestAdoptRequireTrueFromNegation(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&testify.AdoptTestifyRequireTrue{})
	spec.RewriteRun(t,
		test.Golang(`
			package sample

			import "testing"

			func check() bool { return true }

			func TestThing(t *testing.T) {
				ok := check()
				if !ok {
					t.Fatal("not ok")
				}
			}
		`, `
			package sample

			import (
				"testing"
				"github.com/stretchr/testify/require"
			)

			func check() bool { return true }

			func TestThing(t *testing.T) {
				ok := check()
				require.True(t, ok, "not ok")
			}
		`),
	)
}

// `if cond` asserts the condition is false.
func TestAdoptAssertFalseFromCondition(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&testify.AdoptTestifyAssertTrue{})
	spec.RewriteRun(t,
		test.Golang(`
			package sample

			import "testing"

			func check() bool { return false }

			func TestThing(t *testing.T) {
				bad := check()
				if bad {
					t.Errorf("was bad")
				}
			}
		`, `
			package sample

			import (
				"testing"
				"github.com/stretchr/testify/assert"
			)

			func check() bool { return false }

			func TestThing(t *testing.T) {
				bad := check()
				assert.False(t, bad, "was bad")
			}
		`),
	)
}

// A comparison condition is left to the Equal recipe, not converted to True/False.
func TestAdoptRequireTrueNoChangeOnComparison(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&testify.AdoptTestifyRequireTrue{})
	spec.RewriteRun(t,
		test.Golang(`
			package sample

			import "testing"

			func TestThing(t *testing.T) {
				got := 1
				if got != 2 {
					t.Fatal("mismatch")
				}
			}
		`),
	)
}
