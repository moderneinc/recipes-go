/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/simplification"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestPreferBytesContainsRuneNotEqNeg1(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.PreferBytesContainsRune{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "bytes"

			func f(b []byte) bool {
				return bytes.IndexRune(b, 'a') != -1
			}
		`, `
			package main

			import "bytes"

			func f(b []byte) bool {
				return bytes.ContainsRune(b, 'a')
			}
		`),
	)
}

func TestPreferBytesContainsRuneGte0(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.PreferBytesContainsRune{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "bytes"

			func f(b []byte) bool {
				return bytes.IndexRune(b, 'a') >= 0
			}
		`, `
			package main

			import "bytes"

			func f(b []byte) bool {
				return bytes.ContainsRune(b, 'a')
			}
		`),
	)
}

func TestPreferBytesContainsRuneEqNeg1(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.PreferBytesContainsRune{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "bytes"

			func f(b []byte) bool {
				return bytes.IndexRune(b, 'a') == -1
			}
		`, `
			package main

			import "bytes"

			func f(b []byte) bool {
				return !bytes.ContainsRune(b, 'a')
			}
		`),
	)
}

func TestPreferBytesContainsRuneLt0(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.PreferBytesContainsRune{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "bytes"

			func f(b []byte) bool {
				return bytes.IndexRune(b, 'a') < 0
			}
		`, `
			package main

			import "bytes"

			func f(b []byte) bool {
				return !bytes.ContainsRune(b, 'a')
			}
		`),
	)
}

func TestPreferBytesContainsRuneNoChange(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.PreferBytesContainsRune{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "bytes"

			func f(b []byte) int {
				return bytes.IndexRune(b, 'a')
			}
		`),
	)
}
