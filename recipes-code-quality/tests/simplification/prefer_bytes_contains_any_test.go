/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes-code-quality/recipes/simplification"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestPreferBytesContainsAnyNotEqNeg1(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.PreferBytesContainsAny{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "bytes"

			func f(b []byte) bool {
				return bytes.IndexAny(b, "aeiou") != -1
			}
		`, `
			package main

			import "bytes"

			func f(b []byte) bool {
				return bytes.ContainsAny(b, "aeiou")
			}
		`),
	)
}

func TestPreferBytesContainsAnyGte0(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.PreferBytesContainsAny{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "bytes"

			func f(b []byte) bool {
				return bytes.IndexAny(b, "aeiou") >= 0
			}
		`, `
			package main

			import "bytes"

			func f(b []byte) bool {
				return bytes.ContainsAny(b, "aeiou")
			}
		`),
	)
}

func TestPreferBytesContainsAnyEqNeg1(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.PreferBytesContainsAny{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "bytes"

			func f(b []byte) bool {
				return bytes.IndexAny(b, "aeiou") == -1
			}
		`, `
			package main

			import "bytes"

			func f(b []byte) bool {
				return !bytes.ContainsAny(b, "aeiou")
			}
		`),
	)
}

func TestPreferBytesContainsAnyLt0(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.PreferBytesContainsAny{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "bytes"

			func f(b []byte) bool {
				return bytes.IndexAny(b, "aeiou") < 0
			}
		`, `
			package main

			import "bytes"

			func f(b []byte) bool {
				return !bytes.ContainsAny(b, "aeiou")
			}
		`),
	)
}

func TestPreferBytesContainsAnyNoChange(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.PreferBytesContainsAny{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "bytes"

			func f(b []byte) int {
				return bytes.IndexAny(b, "aeiou")
			}
		`),
	)
}
