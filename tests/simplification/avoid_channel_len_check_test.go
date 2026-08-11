/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification_test

import (
	"fmt"
	"testing"

	"github.com/moderneinc/recipes-go/recipes/simplification"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

// racyMarker is the search marker the recipe attaches to a channel length
// check.
const racyMarker = "/*~~(channel length check is racy; the value can change between check and send/receive)~~>*/"

// The length check is flagged regardless of channel direction, comparison
// operator, or which side of the comparison the len call sits on.
func TestAvoidChannelLenCheckFlagged(t *testing.T) {
	cases := []struct {
		name  string
		param string
		expr  string
	}{
		{"equal zero", "ch chan int", "len(ch) == 0"},
		{"greater than zero", "ch chan int", "len(ch) > 0"},
		{"send-only channel", "ch chan<- int", "len(ch) == 0"},
		{"receive-only channel", "ch <-chan int", "len(ch) == 0"},
		{"reversed operand order", "ch chan int", "0 == len(ch)"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			before := fmt.Sprintf(`
				package main

				func f(%s) bool {
					return %s
				}
			`, tc.param, tc.expr)
			after := fmt.Sprintf(`
				package main

				func f(%s) bool {
					return %s%s
				}
			`, tc.param, racyMarker, tc.expr)
			test.NewRecipeSpec().WithRecipe(&simplification.AvoidChannelLenCheck{}).
				RewriteRun(t, test.Golang(before, after))
		})
	}
}

// The non-channel types that len also accepts; none of them should be flagged.
func TestAvoidChannelLenCheckNoChange(t *testing.T) {
	cases := []struct {
		name  string
		param string
		expr  string
	}{
		{"slice", "s []int", "len(s) == 0"},
		{"map", "m map[string]int", "len(m) == 0"},
		{"string", "s string", "len(s) == 0"},
		{"array", "a [3]int", "len(a) == 0"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			src := fmt.Sprintf(`
				package main

				func f(%s) bool {
					return %s
				}
			`, tc.param, tc.expr)
			test.NewRecipeSpec().WithRecipe(&simplification.AvoidChannelLenCheck{}).
				RewriteRun(t, test.Golang(src))
		})
	}
}

// Channels are commonly held as struct fields; the argument to len is then a
// field access rather than a plain identifier.
func TestAvoidChannelLenCheckStructField(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.AvoidChannelLenCheck{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			type S struct {
				ch chan int
			}

			func f(s S) bool {
				return len(s.ch) == 0
			}
		`, fmt.Sprintf(`
			package main

			type S struct {
				ch chan int
			}

			func f(s S) bool {
				return %slen(s.ch) == 0
			}
		`, racyMarker)),
	)
}

// A channel type defined in the same file (`type C chan int`) is resolved via a
// pre-pass over the compilation unit's type declarations and still matched.
func TestAvoidChannelLenCheckNamedChanType(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.AvoidChannelLenCheck{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			type C chan int

			func f(ch C) bool {
				return len(ch) == 0
			}
		`, fmt.Sprintf(`
			package main

			type C chan int

			func f(ch C) bool {
				return %slen(ch) == 0
			}
		`, racyMarker)),
	)
}

// A named type whose underlying type is itself a named channel type
// (`type D C`) is resolved transitively.
func TestAvoidChannelLenCheckTransitiveNamedChanType(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.AvoidChannelLenCheck{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			type C chan int
			type D C

			func f(ch D) bool {
				return len(ch) == 0
			}
		`, fmt.Sprintf(`
			package main

			type C chan int
			type D C

			func f(ch D) bool {
				return %slen(ch) == 0
			}
		`, racyMarker)),
	)
}

// Grouped type declarations (`type ( ... )`) are scanned too.
func TestAvoidChannelLenCheckGroupedNamedChanType(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.AvoidChannelLenCheck{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			type (
				C chan int
				N int
			)

			func f(ch C) bool {
				return len(ch) == 0
			}
		`, fmt.Sprintf(`
			package main

			type (
				C chan int
				N int
			)

			func f(ch C) bool {
				return %slen(ch) == 0
			}
		`, racyMarker)),
	)
}

// A defined non-channel type declared in the same file must not be matched,
// confirming the pre-pass does not over-collect.
func TestAvoidChannelLenCheckNoChangeNamedIntType(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.AvoidChannelLenCheck{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			type Count int

			func f(c Count) bool {
				return c == 0
			}
		`),
	)
}

// Known limitation: a channel type alias (`type D = chan int`) collapses to an
// unknown type at the use site, so it is not recognized as a channel.
func TestAvoidChannelLenCheckNoChangeChanTypeAlias(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.AvoidChannelLenCheck{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			type D = chan int

			func f(ch D) bool {
				return len(ch) == 0
			}
		`),
	)
}

// The per-file pre-pass misses a channel type declared in another file, and the
// no-change result also guards against collected names leaking between files.
func TestAvoidChannelLenCheckNoChangeCrossFileNamedChanType(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.AvoidChannelLenCheck{})
	spec.RewriteRun(t,
		test.GoProject("app",
			test.GoMod(`
				module app

				go 1.21
			`),
			test.Golang(`
				package app

				type C chan int
			`).WithPath("types.go"),
			test.Golang(`
				package app

				func f(ch C) bool {
					return len(ch) == 0
				}
			`).WithPath("use.go"),
		),
	)
}
