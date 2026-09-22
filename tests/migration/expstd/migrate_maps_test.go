/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package expstd_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes/migration/expstd"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func mapsSpec() *test.RecipeSpec {
	return test.NewRecipeSpec().WithRecipe(&expstd.MigrateXExpMapsToStdlib{})
}

// maaslalani/nap collects folder names out of a map. The standard library's
// Keys yields an iterator, so the call is wrapped rather than repointed.
func TestMapsKeysIsCollected(t *testing.T) {
	mapsSpec().RewriteRun(t,
		test.Golang(`
			package main

			import "golang.org/x/exp/maps"

			func folderNames(folders map[string]bool) []string {
				foldersSlice := maps.Keys(folders)
				return foldersSlice
			}
		`, `
			package main

			import (
				"maps"
				"slices"
			)

			func folderNames(folders map[string]bool) []string {
				foldersSlice := slices.Collect(maps.Keys(folders))
				return foldersSlice
			}
		`),
	)
}

// livekit/psrpc reads the handler values out of a map the same way.
func TestMapsValuesIsCollected(t *testing.T) {
	mapsSpec().RewriteRun(t,
		test.Golang(`
			package server

			import "golang.org/x/exp/maps"

			func (s *RPCServer) allHandlers() []Handler {
				return maps.Values(s.handlers)
			}
		`, `
			package server

			import (
				"maps"
				"slices"
			)

			func (s *RPCServer) allHandlers() []Handler {
				return slices.Collect(maps.Values(s.handlers))
			}
		`),
	)
}

// psrpc also spreads the keys into a variadic call; the spread stays outside the
// wrapper. The blank line that separated the standard library group from the
// third-party one is left where it was, since the rewrite only moves the import
// it was asked to.
func TestMapsKeysSpreadIntoVariadic(t *testing.T) {
	mapsSpec().RewriteRun(t,
		test.Golang(`
			package bus

			import (
				"context"

				"golang.org/x/exp/maps"
			)

			func (r *redisBus) sync(ctx context.Context, subscribe map[string]bool) error {
				return r.ps.Subscribe(ctx, maps.Keys(subscribe)...)
			}
		`, `
			package bus

			import (
				"context"

				"maps"
				"slices"
			)

			func (r *redisBus) sync(ctx context.Context, subscribe map[string]bool) error {
				return r.ps.Subscribe(ctx, slices.Collect(maps.Keys(subscribe))...)
			}
		`),
	)
}

// Clear was superseded by the clear builtin, which leaves the maps import
// carrying only the functions that survived unchanged.
func TestMapsClearBecomesBuiltin(t *testing.T) {
	mapsSpec().RewriteRun(t,
		test.Golang(`
			package cache

			import "golang.org/x/exp/maps"

			func reset(m map[string]int, src map[string]int) {
				maps.Clear(m)
				maps.Copy(m, src)
			}
		`, `
			package cache

			import "maps"

			func reset(m map[string]int, src map[string]int) {
				clear(m)
				maps.Copy(m, src)
			}
		`),
	)
}

// Clone, Copy, Equal, EqualFunc and DeleteFunc are identical in the standard
// library, so a file using only those is a path swap.
func TestMapsPureSwap(t *testing.T) {
	mapsSpec().RewriteRun(t,
		test.Golang(`
			package cache

			import "golang.org/x/exp/maps"

			func same(a, b map[string]int) bool {
				c := maps.Clone(a)
				maps.DeleteFunc(c, func(k string, v int) bool { return v == 0 })
				return maps.Equal(c, b)
			}
		`, `
			package cache

			import "maps"

			func same(a, b map[string]int) bool {
				c := maps.Clone(a)
				maps.DeleteFunc(c, func(k string, v int) bool { return v == 0 })
				return maps.Equal(c, b)
			}
		`),
	)
}

// The iterator forms arrived in Go 1.23; on an older module the file is left
// whole rather than half migrated.
func TestMapsKeysBlockedBeforeGo123(t *testing.T) {
	mapsSpec().RewriteRun(t,
		test.GoProject("app",
			test.GoMod(`
				module example.com/app

				go 1.22

				require golang.org/x/exp v0.0.0-20240103183307-be819d1f06fc
			`),
			test.Golang(`
				package app

				import "golang.org/x/exp/maps"

				func keys(m map[string]int) []string {
					return maps.Keys(m)
				}
			`),
		),
	)
}

// A file using only the unchanged functions still migrates on Go 1.22.
func TestMapsPureSwapOnGo122(t *testing.T) {
	mapsSpec().RewriteRun(t,
		test.GoProject("app",
			test.GoMod(`
				module example.com/app

				go 1.22

				require golang.org/x/exp v0.0.0-20240103183307-be819d1f06fc
			`),
			test.Golang(`
				package app

				import "golang.org/x/exp/maps"

				func clone(m map[string]int) map[string]int {
					return maps.Clone(m)
				}
			`, `
				package app

				import "maps"

				func clone(m map[string]int) map[string]int {
					return maps.Clone(m)
				}
			`),
		),
	)
}

// A file binding `slices` to something else would resolve the emitted wrapper
// wrongly, so it is left alone.
func TestMapsBlockedByShadowedSlicesName(t *testing.T) {
	mapsSpec().RewriteRun(t,
		test.Golang(`
			package app

			import (
				slices "example.com/app/internal/slices"

				"golang.org/x/exp/maps"
			)

			func keys(m map[string]int) []string {
				_ = slices.Helper
				return maps.Keys(m)
			}
		`),
	)
}

// A `for _, k := range maps.Keys(m)` ranges the standard library's iterator
// directly rather than collecting it into a slice first, and the blank index the
// slice form needed goes away. go-acme/lego writes the migrated shape already.
func TestMapsKeysRangedDirectly(t *testing.T) {
	mapsSpec().RewriteRun(t,
		test.Golang(`
			package resolver

			import "golang.org/x/exp/maps"

			func (s *solverManager) names(solvers map[string]solver) []string {
				var out []string
				for _, k := range maps.Keys(solvers) {
					out = append(out, k)
				}
				return out
			}
		`, `
			package resolver

			import "maps"

			func (s *solverManager) names(solvers map[string]solver) []string {
				var out []string
				for k := range maps.Keys(solvers) {
					out = append(out, k)
				}
				return out
			}
		`),
	)
}

// A single-variable range means something different on each side: over the
// x/exp slice it yields indices, over the iterator it would yield keys. That one
// keeps the slice.
func TestMapsKeysSingleVariableRangeIsCollected(t *testing.T) {
	mapsSpec().RewriteRun(t,
		test.Golang(`
			package app

			import "golang.org/x/exp/maps"

			func indexes(m map[string]int) []int {
				var out []int
				for i := range maps.Keys(m) {
					out = append(out, i)
				}
				return out
			}
		`, `
			package app

			import (
				"maps"
				"slices"
			)

			func indexes(m map[string]int) []int {
				var out []int
				for i := range slices.Collect(maps.Keys(m)) {
					out = append(out, i)
				}
				return out
			}
		`),
	)
}

// A range that reads the index keeps the slice too.
func TestMapsKeysIndexedRangeIsCollected(t *testing.T) {
	mapsSpec().RewriteRun(t,
		test.Golang(`
			package app

			import "golang.org/x/exp/maps"

			func numbered(m map[string]int) []string {
				var out []string
				for i, k := range maps.Keys(m) {
					_ = i
					out = append(out, k)
				}
				return out
			}
		`, `
			package app

			import (
				"maps"
				"slices"
			)

			func numbered(m map[string]int) []string {
				var out []string
				for i, k := range slices.Collect(maps.Keys(m)) {
					_ = i
					out = append(out, k)
				}
				return out
			}
		`),
	)
}

// coroot/coroot collects the keys and sorts them on the next line, which is
// exactly what slices.Sorted does in one step.
func TestMapsKeysSortedCollapses(t *testing.T) {
	mapsSpec().RewriteRun(t,
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
	)
}

// A sort.Slice with its own comparator is not an ascending sort of an ordered
// element type, so the collapse does not apply and the keys are collected.
func TestMapsValuesWithCustomSortIsCollected(t *testing.T) {
	mapsSpec().RewriteRun(t,
		test.Golang(`
			package cache

			import (
				"sort"

				"golang.org/x/exp/maps"
			)

			func ordered(m map[string]chunk) []chunk {
				chunks := maps.Values(m)
				sort.Slice(chunks, func(i, j int) bool {
					return chunks[i].Created < chunks[j].Created
				})
				return chunks
			}
		`, `
			package cache

			import (
				"sort"

				"maps"
				"slices"
			)

			func ordered(m map[string]chunk) []chunk {
				chunks := slices.Collect(maps.Values(m))
				sort.Slice(chunks, func(i, j int) bool {
					return chunks[i].Created < chunks[j].Created
				})
				return chunks
			}
		`),
	)
}

// A sort of some other variable does not license the collapse.
func TestMapsKeysSortOfOtherVariableIsCollected(t *testing.T) {
	mapsSpec().RewriteRun(t,
		test.Golang(`
			package app

			import (
				"sort"

				"golang.org/x/exp/maps"
			)

			func names(m map[string]int, other []string) []string {
				keys := maps.Keys(m)
				sort.Strings(other)
				return keys
			}
		`, `
			package app

			import (
				"sort"

				"maps"
				"slices"
			)

			func names(m map[string]int, other []string) []string {
				keys := slices.Collect(maps.Keys(m))
				sort.Strings(other)
				return keys
			}
		`),
	)
}
