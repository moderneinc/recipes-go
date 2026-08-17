/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package testify_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes/migration/testify"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestAdoptAssertNoErrorFromErrorf(t *testing.T) {
	// given
	spec := test.NewRecipeSpec().WithRecipe(&testify.AdoptTestifyAssertNoError{})

	// when / then a non-fatal reporter maps to assert
	spec.RewriteRun(t,
		test.Golang(`
			package sample

			import "testing"

			func do() error { return nil }

			func TestThing(t *testing.T) {
				err := do()
				if err != nil {
					t.Errorf("unexpected: %v", err)
				}
			}
		`, `
			package sample

			import (
				"testing"
				"github.com/stretchr/testify/assert"
			)

			func do() error { return nil }

			func TestThing(t *testing.T) {
				err := do()
				assert.NoError(t, err, "unexpected")
			}
		`),
	)
}

// The fatal reporter is out of scope for the assert recipe (require handles it).
func TestAdoptAssertNoErrorNoChangeOnFatal(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&testify.AdoptTestifyAssertNoError{})
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
