/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package testify_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes/migration/testify"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestAdoptRequireNoErrorFatal(t *testing.T) {
	// given
	spec := test.NewRecipeSpec().WithRecipe(&testify.AdoptTestifyRequireNoError{})

	// when / then
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
		`, `
			package sample

			import (
				"testing"

				"github.com/stretchr/testify/require"
			)

			func do() error { return nil }

			func TestThing(t *testing.T) {
				err := do()
				require.NoError(t, err)
			}
		`),
	)
}

func TestAdoptRequireNoErrorFatalf(t *testing.T) {
	// given
	spec := test.NewRecipeSpec().WithRecipe(&testify.AdoptTestifyRequireNoError{})

	// when / then
	spec.RewriteRun(t,
		test.Golang(`
			package sample

			import "testing"

			func do() error { return nil }

			func TestThing(t *testing.T) {
				err := do()
				if err != nil {
					t.Fatalf("unexpected: %v", err)
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
				require.NoError(t, err, "unexpected")
			}
		`),
	)
}

// The subtest receiver name carries through to the generated call.
func TestAdoptRequireNoErrorSubtestReceiver(t *testing.T) {
	// given
	spec := test.NewRecipeSpec().WithRecipe(&testify.AdoptTestifyRequireNoError{})

	// when / then
	spec.RewriteRun(t,
		test.Golang(`
			package sample

			import "testing"

			func do() error { return nil }

			func TestThing(t *testing.T) {
				t.Run("x", func(tt *testing.T) {
					err := do()
					if err != nil {
						tt.Fatal(err)
					}
				})
			}
		`, `
			package sample

			import (
				"testing"

				"github.com/stretchr/testify/require"
			)

			func do() error { return nil }

			func TestThing(t *testing.T) {
				t.Run("x", func(tt *testing.T) {
					err := do()
					require.NoError(tt, err)
				})
			}
		`),
	)
}

// The inline-init form inlines the call: `if err := do(); err != nil { ... }`.
func TestAdoptRequireNoErrorInlineInit(t *testing.T) {
	// given
	spec := test.NewRecipeSpec().WithRecipe(&testify.AdoptTestifyRequireNoError{})

	// when / then
	spec.RewriteRun(t,
		test.Golang(`
			package sample

			import "testing"

			func do() error { return nil }

			func TestThing(t *testing.T) {
				if err := do(); err != nil {
					t.Fatalf("boom: %v", err)
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
				require.NoError(t, do(), "boom")
			}
		`),
	)
}

// The `=` assignment form (not `:=`) reassigns an outer variable, so inlining
// would drop that assignment; it must be left unchanged.
func TestAdoptRequireNoErrorNoChangeInlineAssignment(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&testify.AdoptTestifyRequireNoError{})
	spec.RewriteRun(t,
		test.Golang(`
			package sample

			import "testing"

			func do() error { return nil }

			func TestThing(t *testing.T) {
				var err error
				if err = do(); err != nil {
					t.Fatal(err)
				}
			}
		`),
	)
}

// A non-fatal reporter (t.Error) is left for the assert-variant recipe.
func TestAdoptRequireNoErrorNoChangeOnError(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&testify.AdoptTestifyRequireNoError{})
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
		`),
	)
}

// log.Fatal is not a test assertion and must not be rewritten.
func TestAdoptRequireNoErrorNoChangeLogFatal(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&testify.AdoptTestifyRequireNoError{})
	spec.RewriteRun(t,
		test.Golang(`
			package sample

			import "log"

			func do() error { return nil }

			func run() {
				err := do()
				if err != nil {
					log.Fatal(err)
				}
			}
		`),
	)
}

// A message that references a context variable (here `dir`) keeps it: the
// redundant `%v` of err is stripped, but `in %s` naming `dir` is preserved via
// the f-variant, so `dir` stays used.
func TestAdoptRequireNoErrorPreservesContextMessage(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&testify.AdoptTestifyRequireNoError{})
	spec.RewriteRun(t,
		test.Golang(`
			package sample

			import "testing"

			func do() error { return nil }

			func TestThing(t *testing.T) {
				for _, dir := range []string{"a"} {
					err := do()
					if err != nil {
						t.Fatalf("in %s: %v", dir, err)
					}
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
				for _, dir := range []string{"a"} {
					err := do()
					require.NoErrorf(t, err, "in %s", dir)
				}
			}
		`),
	)
}

// A guard whose body does more than report must not collapse to a single call.
func TestAdoptRequireNoErrorNoChangeExtraStatement(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&testify.AdoptTestifyRequireNoError{})
	spec.RewriteRun(t,
		test.Golang(`
			package sample

			import "testing"

			func do() error { return nil }

			func TestThing(t *testing.T) {
				err := do()
				if err != nil {
					t.Log("cleanup")
					t.Fatal(err)
				}
			}
		`),
	)
}
