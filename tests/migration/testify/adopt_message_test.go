/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package testify_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes/migration/testify"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

// The descriptive part of the message is kept; the trailing `, got %+v` dump of
// the asserted value is dropped.
func TestPreserveMessageDropsGotClauseNil(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&testify.AdoptTestifyAssertNil{})
	spec.RewriteRun(t,
		test.Golang(`
			package sample

			import "testing"

			type prepared struct {
				DelegatesTo any
				RecipeList  []int
			}

			func TestThing(t *testing.T) {
				pr := prepared{}
				if pr.DelegatesTo != nil {
					t.Errorf("the composite itself should not delegate, got %+v", pr.DelegatesTo)
				}
			}
		`, `
			package sample

			import (
				"testing"
				"github.com/stretchr/testify/assert"
			)

			type prepared struct {
				DelegatesTo any
				RecipeList  []int
			}

			func TestThing(t *testing.T) {
				pr := prepared{}
				assert.Nil(t, pr.DelegatesTo, "the composite itself should not delegate")
			}
		`),
	)
}

// The `, got %d: %+v` value dump is dropped from a Len message, keeping the
// description.
func TestPreserveMessageDropsGotClauseLen(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&testify.AdoptTestifyRequireLen{})
	spec.RewriteRun(t,
		test.Golang(`
			package sample

			import "testing"

			type prepared struct {
				RecipeList []int
			}

			func TestThing(t *testing.T) {
				pr := prepared{}
				if len(pr.RecipeList) != 1 {
					t.Fatalf("expected 1 prepared child, got %d: %+v", len(pr.RecipeList), pr.RecipeList)
				}
			}
		`, `
			package sample

			import (
				"testing"
				"github.com/stretchr/testify/require"
			)

			type prepared struct {
				RecipeList []int
			}

			func TestThing(t *testing.T) {
				pr := prepared{}
				require.Len(t, pr.RecipeList, 1, "expected 1 prepared child")
			}
		`),
	)
}

// Stripping the value-dump must not leave an unbalanced parenthesis: the `)`
// dropped with `: got %d` is restored.
func TestPreserveMessageBalancesParens(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&testify.AdoptTestifyRequireLen{})
	spec.RewriteRun(t,
		test.Golang(`
			package sample

			import "testing"

			func build() []int { return nil }

			func TestThing(t *testing.T) {
				s := build()
				if len(s) != 2 {
					t.Fatalf("want 2 entries (malformed skipped): got %d", len(s))
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
				require.Len(t, s, 2, "want 2 entries (malformed skipped)")
			}
		`),
	)
}

// When the format cannot be parsed with confidence (here an indexed verb), the
// original template is kept verbatim.
func TestPreserveMessageKeepsComplexFormat(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&testify.AdoptTestifyRequireNoError{})
	spec.RewriteRun(t,
		test.Golang(`
			package sample

			import "testing"

			func do() error { return nil }

			func TestThing(t *testing.T) {
				attempt := 2
				err := do()
				if err != nil {
					t.Fatalf("attempt %[1]d failed: %v", attempt, err)
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
				attempt := 2
				err := do()
				require.NoErrorf(t, err, "attempt %[1]d failed: %v", attempt, err)
			}
		`),
	)
}
