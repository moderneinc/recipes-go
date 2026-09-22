/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package expstd_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes/migration/expstd"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func slicesSpec() *test.RecipeSpec {
	return test.NewRecipeSpec().WithRecipe(&expstd.MigrateXExpSlicesToStdlib{})
}

// fiatjaf/nak passes a comparator both ways a current caller can — named, and
// spelled out returning an int. Neither is the pre-2023 boolean form, so the
// whole file is a path swap.
func TestSlicesSortWithNamedComparator(t *testing.T) {
	slicesSpec().RewriteRun(t,
		test.Golang(`
			package nak

			import (
				"fmt"

				"golang.org/x/exp/slices"
			)

			func order(events []Event, commits []Commit) {
				slices.SortFunc(events, CompareRelayEvent)
				slices.SortStableFunc(commits, func(a, b Commit) int {
					return int(a.When - b.When)
				})
				fmt.Println(slices.Contains(nil, ""))
			}
		`, `
			package nak

			import (
				"fmt"

				"slices"
			)

			func order(events []Event, commits []Commit) {
				slices.SortFunc(events, CompareRelayEvent)
				slices.SortStableFunc(commits, func(a, b Commit) int {
					return int(a.When - b.When)
				})
				fmt.Println(slices.Contains(nil, ""))
			}
		`),
	)
}

// The pre-2023 x/exp comparator returned bool. Neither the current x/exp nor the
// standard library accepts it, so the file is left for a hand conversion.
func TestSlicesBlockedByBooleanComparator(t *testing.T) {
	slicesSpec().RewriteRun(t,
		test.Golang(`
			package legacy

			import "golang.org/x/exp/slices"

			type item struct{ weight int }

			func sortItems(items []item) {
				slices.SortFunc(items, func(a, b item) bool { return a.weight < b.weight })
			}
		`),
	)
}

// The module targets a release older than the one that added stdlib slices.
func TestSlicesBlockedByGoVersion(t *testing.T) {
	slicesSpec().RewriteRun(t,
		test.GoProject("app",
			test.GoMod(`
				module example.com/app

				go 1.20

				require golang.org/x/exp v0.0.0-20240103183307-be819d1f06fc
			`),
			test.Golang(`
				package app

				import "golang.org/x/exp/slices"

				func has(s []string, v string) bool {
					return slices.Contains(s, v)
				}
			`),
		),
	)
}

// The same file on Go 1.21 migrates.
func TestSlicesMigratesOnGo121(t *testing.T) {
	slicesSpec().RewriteRun(t,
		test.GoProject("app",
			test.GoMod(`
				module example.com/app

				go 1.21

				require golang.org/x/exp v0.0.0-20240103183307-be819d1f06fc
			`),
			test.Golang(`
				package app

				import "golang.org/x/exp/slices"

				func has(s []string, v string) bool {
					return slices.Contains(s, v)
				}
			`, `
				package app

				import "slices"

				func has(s []string, v string) bool {
					return slices.Contains(s, v)
				}
			`),
		),
	)
}

// Both packages imported: they bind the same name, so the file is left alone.
func TestSlicesBlockedWhenStdlibAlreadyImported(t *testing.T) {
	slicesSpec().RewriteRun(t,
		test.Golang(`
			package app

			import (
				"slices"

				expslices "golang.org/x/exp/slices"
			)

			func has(s []string, v string) bool {
				return slices.Contains(s, v) && expslices.Contains(s, v)
			}
		`),
	)
}
