/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package expstd_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes/migration/expstd"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func constraintsSpec() *test.RecipeSpec {
	return test.NewRecipeSpec().WithRecipe(&expstd.MigrateXExpConstraintsToStdlib{})
}

// Ordered is the one constraint the standard library took, as cmp.Ordered.
func TestConstraintsOrderedBecomesCmpOrdered(t *testing.T) {
	constraintsSpec().RewriteRun(t,
		test.Golang(`
			package numeric

			import "golang.org/x/exp/constraints"

			func Clamp[T constraints.Ordered](v, lo, hi T) T {
				if v < lo {
					return lo
				}
				if v > hi {
					return hi
				}
				return v
			}
		`, `
			package numeric

			import "cmp"

			func Clamp[T cmp.Ordered](v, lo, hi T) T {
				if v < lo {
					return lo
				}
				if v > hi {
					return hi
				}
				return v
			}
		`),
	)
}

// An aliased import keeps its name, so only the path moves.
func TestConstraintsAliasedImport(t *testing.T) {
	constraintsSpec().RewriteRun(t,
		test.Golang(`
			package numeric

			import cons "golang.org/x/exp/constraints"

			func Max[T cons.Ordered](a, b T) T {
				if a > b {
					return a
				}
				return b
			}
		`, `
			package numeric

			import cons "cmp"

			func Max[T cons.Ordered](a, b T) T {
				if a > b {
					return a
				}
				return b
			}
		`),
	)
}

// The numeric constraints never landed in the standard library, so a file naming
// one keeps its x/exp import.
func TestConstraintsBlockedByIntegerConstraint(t *testing.T) {
	constraintsSpec().RewriteRun(t,
		test.Golang(`
			package numeric

			import "golang.org/x/exp/constraints"

			func Sum[T constraints.Integer](vs []T) T {
				var total T
				for _, v := range vs {
					total += v
				}
				return total
			}

			func Max[T constraints.Ordered](a, b T) T {
				if a > b {
					return a
				}
				return b
			}
		`),
	)
}

// A file already binding `cmp` would collide with the emitted reference.
func TestConstraintsBlockedByExistingCmpBinding(t *testing.T) {
	constraintsSpec().RewriteRun(t,
		test.Golang(`
			package numeric

			import (
				"cmp"

				"golang.org/x/exp/constraints"
			)

			func Max[T constraints.Ordered](a, b T) T {
				return cmp.Or(a, b)
			}
		`),
	)
}

// Code-Hex/go-generics-cache builds a union out of the numeric constraints. None
// of them reached the standard library, so the union keeps the x/exp import.
func TestConstraintsBlockedByNumericUnion(t *testing.T) {
	constraintsSpec().RewriteRun(t,
		test.Golang(`
			package cache

			import "golang.org/x/exp/constraints"

			type Number interface {
				constraints.Integer | constraints.Float | constraints.Complex
			}
		`),
	)
}
