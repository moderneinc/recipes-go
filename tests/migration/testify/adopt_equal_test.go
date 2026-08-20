/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package testify_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes/migration/testify"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

// `got != want` becomes require.Equal with the want operand ordered first.
func TestAdoptRequireEqualFromNotEqual(t *testing.T) {
	// given
	spec := test.NewRecipeSpec().WithRecipe(&testify.AdoptTestifyRequireEqual{})

	// when / then
	spec.RewriteRun(t,
		test.Golang(`
			package sample

			import "testing"

			func TestThing(t *testing.T) {
				got := 1
				want := 2
				if got != want {
					t.Fatalf("got %v, want %v", got, want)
				}
			}
		`, `
			package sample

			import (
				"testing"

				"github.com/stretchr/testify/require"
			)

			func TestThing(t *testing.T) {
				got := 1
				want := 2
				require.Equal(t, want, got)
			}
		`),
	)
}

// `got == want` guarded by a non-fatal reporter becomes assert.NotEqual.
func TestAdoptAssertNotEqualFromEqual(t *testing.T) {
	// given
	spec := test.NewRecipeSpec().WithRecipe(&testify.AdoptTestifyAssertEqual{})

	// when / then
	spec.RewriteRun(t,
		test.Golang(`
			package sample

			import "testing"

			func TestThing(t *testing.T) {
				got := 1
				want := 2
				if got == want {
					t.Errorf("did not expect %v", got)
				}
			}
		`, `
			package sample

			import (
				"testing"

				"github.com/stretchr/testify/assert"
			)

			func TestThing(t *testing.T) {
				got := 1
				want := 2
				assert.NotEqual(t, want, got, "did not expect")
			}
		`),
	)
}

// A literal operand is treated as the expected value and ordered first.
func TestAdoptRequireEqualLiteralExpectedFirst(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&testify.AdoptTestifyRequireEqual{})
	spec.RewriteRun(t,
		test.Golang(`
			package sample

			import "testing"

			func TestThing(t *testing.T) {
				n := 3
				if n != 42 {
					t.Fatalf("n = %d", n)
				}
			}
		`, `
			package sample

			import (
				"testing"

				"github.com/stretchr/testify/require"
			)

			func TestThing(t *testing.T) {
				n := 3
				require.Equal(t, 42, n, "n")
			}
		`),
	)
}

// Pointer identity comparison must not convert: Go `==` is identity but
// testify Equal/NotEqual is value equality (reflect.DeepEqual), which differ.
func TestAdoptRequireEqualNoChangeOnPointerIdentity(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&testify.AdoptTestifyRequireEqual{})
	spec.RewriteRun(t,
		test.Golang(`
			package sample

			import "testing"

			type box struct{ v int }

			func TestThing(t *testing.T) {
				a := &box{1}
				b := &box{1}
				if a == b {
					t.Fatal("must be distinct instances")
				}
			}
		`),
	)
}

// A nil comparison is a presence check (NoError/Error territory), not equality.
func TestAdoptRequireEqualNoChangeOnNil(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&testify.AdoptTestifyRequireEqual{})
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

// A message with contextual args keeps them (and the f-variant): the redundant
// `got`/`want` value-dump is stripped, but the `%s` naming `name` is preserved.
func TestAdoptRequireEqualPreservesContextMessage(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&testify.AdoptTestifyRequireEqual{})
	spec.RewriteRun(t,
		test.Golang(`
			package sample

			import "testing"

			func TestThing(t *testing.T) {
				for _, name := range []string{"a"} {
					got := 1
					want := 2
					if got != want {
						t.Fatalf("%s: got %v, want %v", name, got, want)
					}
				}
			}
		`, `
			package sample

			import (
				"testing"

				"github.com/stretchr/testify/require"
			)

			func TestThing(t *testing.T) {
				for _, name := range []string{"a"} {
					got := 1
					want := 2
					require.Equalf(t, want, got, "%s", name)
				}
			}
		`),
	)
}
