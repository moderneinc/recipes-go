/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package expstd_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes/migration/expstd"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

// The whole module moves and the x/exp requirement goes with it.
func TestMigrateWholeModule(t *testing.T) {
	test.NewRecipeSpec().WithRecipe(&expstd.MigrateXExpToStdlib{}).RewriteRun(t,
		test.GoProject("psrpc",
			test.GoMod(`
				module example.com/psrpc

				go 1.23

				require (
					github.com/redis/go-redis/v9 v9.7.0
					golang.org/x/exp v0.0.0-20240103183307-be819d1f06fc
				)
			`, `
				module example.com/psrpc

				go 1.23

				require (
					github.com/redis/go-redis/v9 v9.7.0
				)
			`),
			test.Golang(`
				package server

				import "golang.org/x/exp/maps"

				func handlers(m map[string]int) []int {
					return maps.Values(m)
				}
			`, `
				package server

				import (
					"maps"
					"slices"
				)

				func handlers(m map[string]int) []int {
					return slices.Collect(maps.Values(m))
				}
			`),
			test.Golang(`
				package request

				import "golang.org/x/exp/slices"

				func has(s []string, v string) bool {
					return slices.Contains(s, v)
				}
			`, `
				package request

				import "slices"

				func has(s []string, v string) bool {
					return slices.Contains(s, v)
				}
			`),
		),
	)
}

// A file the migration cannot touch keeps the module on x/exp, so the
// requirement stays.
func TestMigrateKeepsDependencyWhenAFileIsBlocked(t *testing.T) {
	test.NewRecipeSpec().WithRecipe(&expstd.MigrateXExpToStdlib{}).RewriteRun(t,
		test.GoProject("app",
			test.GoMod(`
				module example.com/app

				go 1.23

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
			`),
		),
	)
}

// An indirect requirement pins a version the module graph asked for, so it is
// left alone even with nothing importing it.
func TestMigrateKeepsIndirectRequirement(t *testing.T) {
	test.NewRecipeSpec().WithRecipe(&expstd.RemoveXExpDependency{}).RewriteRun(t,
		test.GoProject("app",
			test.GoMod(`
				module example.com/app

				go 1.23

				require (
					golang.org/x/exp v0.0.0-20240103183307-be819d1f06fc // indirect
				)
			`),
			test.Golang(`
				package app

				import "slices"

				func has(s []string, v string) bool {
					return slices.Contains(s, v)
				}
			`),
		),
	)
}

// livekit/psrpc's shape: the requirement is dropped from source that has not
// been migrated yet, since the scan asks whether the migration empties each
// file rather than whether it already has. The Moderne CLI scans before a
// recipe list's edits are applied, so a decision read off the post-migration
// imports never fires there.
func TestMigrateDropsDependencyAheadOfSourceMigration(t *testing.T) {
	test.NewRecipeSpec().WithRecipe(&expstd.RemoveXExpDependency{}).RewriteRun(t,
		test.GoProject("psrpc",
			test.GoMod(`
				module example.com/psrpc

				go 1.23

				require (
					github.com/redis/go-redis/v9 v9.7.0
					golang.org/x/exp v0.0.0-20240103183307-be819d1f06fc
				)
			`, `
				module example.com/psrpc

				go 1.23

				require (
					github.com/redis/go-redis/v9 v9.7.0
				)
			`),
			test.Golang(`
				package bus

				import "golang.org/x/exp/maps"

				func names(m map[string]int) []string {
					return maps.Keys(m)
				}
			`),
		),
	)
}

// An x/exp package no recipe covers keeps the requirement, however much of the
// rest moves.
func TestMigrateKeepsDependencyForUncoveredPackage(t *testing.T) {
	test.NewRecipeSpec().WithRecipe(&expstd.RemoveXExpDependency{}).RewriteRun(t,
		test.GoProject("app",
			test.GoMod(`
				module example.com/app

				go 1.23

				require golang.org/x/exp v0.0.0-20240103183307-be819d1f06fc
			`),
			test.Golang(`
				package app

				import "golang.org/x/exp/slices"

				func has(s []string, v string) bool {
					return slices.Contains(s, v)
				}
			`),
			test.Golang(`
				package app

				import "golang.org/x/exp/rand"

				func roll() uint64 {
					return rand.Uint64()
				}
			`),
		),
	)
}

// coroot/coroot's shape: both files collect the keys and sort them on the next
// line, which collapses to slices.Sorted. One of them sorts with x/exp's own
// slices. The composite migrates slices first, so by the
// time the maps rewrite runs the file binds the standard library's slices and
// the wrapper it emits resolves to it. The blank line that separated the
// standard library group from the third-party one stays where it was, since the
// rewrite only moves the imports it was asked to.
func TestMigrateKeysSortedWithXExpSlices(t *testing.T) {
	test.NewRecipeSpec().WithRecipe(&expstd.MigrateXExpToStdlib{}).RewriteRun(t,
		test.GoProject("coroot",
			test.GoMod(`
				module example.com/coroot

				go 1.25

				require golang.org/x/exp v0.0.0-20240103183307-be819d1f06fc
			`, `
				module example.com/coroot

				go 1.25
			`),
			test.Golang(`
				package db

				import (
					"golang.org/x/exp/maps"
					"golang.org/x/exp/slices"
				)

				func categoryNames(settings map[string]*Category) []string {
					names := maps.Keys(settings)
					slices.Sort(names)
					return names
				}
			`, `
				package db

				import (
					"maps"
					"slices"
				)

				func categoryNames(settings map[string]*Category) []string {
					names := slices.Sorted(maps.Keys(settings))
					return names
				}
			`),
			test.Golang(`
				package utils

				import (
					"sort"

					"golang.org/x/exp/maps"
				)

				func (ss *StringSet) Items() []string {
					res := maps.Keys(ss.m)
					sort.Strings(res)
					return res
				}
			`, `
				package utils

				import (
					"sort"

					"maps"
					"slices"
				)

				func (ss *StringSet) Items() []string {
					res := slices.Sorted(maps.Keys(ss.m))
					return res
				}
			`),
		),
	)
}
