/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package testify_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes/migration/testify"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestAdoptRequireErrorFromFatal(t *testing.T) {
	// given
	spec := test.NewRecipeSpec().WithRecipe(&testify.AdoptTestifyRequireError{})

	// when / then `err == nil` guarded by a fatal reporter becomes require.Error
	spec.RewriteRun(t,
		test.Golang(`
			package sample

			import "testing"

			func do() error { return nil }

			func TestThing(t *testing.T) {
				err := do()
				if err == nil {
					t.Fatal("expected an error")
				}
			}
		`, `
			package sample

			import (
				"testing"
				"github.com/stretchr/testify/require"
			)

			func do() error { return nil }

			func TestThing(t *testing.T) {
				err := do()
				require.Error(t, err, "expected an error")
			}
		`),
	)
}

func TestAdoptAssertErrorFromError(t *testing.T) {
	// given
	spec := test.NewRecipeSpec().WithRecipe(&testify.AdoptTestifyAssertError{})

	// when / then `err == nil` guarded by a non-fatal reporter becomes assert.Error
	spec.RewriteRun(t,
		test.Golang(`
			package sample

			import "testing"

			func do() error { return nil }

			func TestThing(t *testing.T) {
				err := do()
				if err == nil {
					t.Error("expected an error")
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
				assert.Error(t, err, "expected an error")
			}
		`),
	)
}

// require.Error must not touch a `!= nil` guard (that is NoError's job).
func TestAdoptRequireErrorNoChangeOnNotNil(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&testify.AdoptTestifyRequireError{})
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
