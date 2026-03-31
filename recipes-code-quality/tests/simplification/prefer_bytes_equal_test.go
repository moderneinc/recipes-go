/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/simplification"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestPreferBytesEqualPositive(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.PreferBytesEqual{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "bytes"

			func f(a, b []byte) bool {
				return bytes.Compare(a, b) == 0
			}
		`, `
			package main

			import "bytes"

			func f(a, b []byte) bool {
				return bytes.Equal(a, b)
			}
		`),
	)
}

func TestPreferBytesEqualNegative(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.PreferBytesEqual{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "bytes"

			func f(a, b []byte) bool {
				return bytes.Compare(a, b) != 0
			}
		`, `
			package main

			import "bytes"

			func f(a, b []byte) bool {
				return !bytes.Equal(a, b)
			}
		`),
	)
}

func TestPreferBytesEqualNoChange(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.PreferBytesEqual{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "bytes"

			func f(a, b []byte) int {
				return bytes.Compare(a, b)
			}
		`),
	)
}
