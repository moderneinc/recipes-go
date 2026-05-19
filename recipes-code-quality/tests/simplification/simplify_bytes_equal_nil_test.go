/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes-code-quality/recipes/simplification"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestSimplifyBytesEqualNilRight(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.SimplifyBytesEqualNil{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "bytes"

			func f(b []byte) bool {
				return bytes.Equal(b, nil)
			}
		`, `
			package main

			import "bytes"

			func f(b []byte) bool {
				return len(b) == 0
			}
		`),
	)
}

func TestSimplifyBytesEqualNilLeft(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.SimplifyBytesEqualNil{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "bytes"

			func f(b []byte) bool {
				return bytes.Equal(nil, b)
			}
		`, `
			package main

			import "bytes"

			func f(b []byte) bool {
				return len(b) == 0
			}
		`),
	)
}

func TestSimplifyBytesEqualNilNoChange(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.SimplifyBytesEqualNil{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "bytes"

			func f(a, b []byte) bool {
				return bytes.Equal(a, b)
			}
		`),
	)
}
