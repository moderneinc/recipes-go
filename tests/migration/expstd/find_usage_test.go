/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package expstd_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes/migration/expstd"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func findSpec() *test.RecipeSpec {
	return test.NewRecipeSpec().WithRecipe(&expstd.FindXExpUsage{})
}

// Each call carries its standard-library form, and the shapes without one are
// warnings rather than suggestions.
func TestFindXExpUsageReportsReplacements(t *testing.T) {
	findSpec().RewriteRun(t,
		test.Golang(`
			package app

			import (
				"golang.org/x/exp/constraints"
				"golang.org/x/exp/maps"
				"golang.org/x/exp/slices"
			)

			func keys(m map[string]int) []string {
				return maps.Keys(m)
			}

			func has(s []string, v string) bool {
				return slices.Contains(s, v)
			}

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
		`, `
			package app

			import (
				"golang.org/x/exp/constraints"
				"golang.org/x/exp/maps"
				"golang.org/x/exp/slices"
			)

			func keys(m map[string]int) []string {
				return /*~~(maps.Keys returns an iter.Seq in the standard library; wrap it as slices.Collect(maps.Keys(m)) (Go 1.23))~~>*/maps.Keys(m)
			}

			func has(s []string, v string) bool {
				return /*~~(replaceable with slices.Contains from the standard library (Go 1.21))~~>*/slices.Contains(s, v)
			}

			func Sum[T /*~~(constraints.Integer has no standard-library counterpart; declare the type set in your own package to drop golang.org/x/exp)~~>*/constraints.Integer](vs []T) T {
				var total T
				for _, v := range vs {
					total += v
				}
				return total
			}

			func Max[T /*~~(replaceable with cmp.Ordered from the standard library (Go 1.21))~~>*/constraints.Ordered](a, b T) T {
				if a > b {
					return a
				}
				return b
			}
		`),
	)
}

// A boolean comparator is reported as needing a hand conversion, not as a
// mechanical replacement.
func TestFindXExpUsageWarnsOnBooleanComparator(t *testing.T) {
	findSpec().RewriteRun(t,
		test.Golang(`
			package legacy

			import "golang.org/x/exp/slices"

			type item struct{ weight int }

			func sortItems(items []item) {
				slices.SortFunc(items, func(a, b item) bool { return a.weight < b.weight })
			}
		`, `
			package legacy

			import "golang.org/x/exp/slices"

			type item struct{ weight int }

			func sortItems(items []item) {
				/*~~(slices.SortFunc takes a three-way `+"`cmp func(a, b E) int`"+`; this boolean comparator is the pre-2023 x/exp signature and has to be converted by hand)~~>*/slices.SortFunc(items, func(a, b item) bool { return a.weight < b.weight })
			}
		`),
	)
}

// No x/exp import: nothing reported.
func TestFindXExpUsageQuietOnStdlibFile(t *testing.T) {
	findSpec().RewriteRun(t,
		test.Golang(`
			package app

			import "slices"

			func has(s []string, v string) bool {
				return slices.Contains(s, v)
			}
		`),
	)
}
