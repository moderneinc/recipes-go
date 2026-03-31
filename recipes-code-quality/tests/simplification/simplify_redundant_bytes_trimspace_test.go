/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/simplification"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestSimplifyRedundantBytesTrimSpace(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.SimplifyRedundantBytesTrimSpace{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "bytes"

			func f(b []byte) []byte {
				return bytes.TrimSpace(bytes.TrimSpace(b))
			}
		`, `
			package main

			import "bytes"

			func f(b []byte) []byte {
				return bytes.TrimSpace(b)
			}
		`),
	)
}

func TestSimplifyRedundantBytesTrimSpaceNoChange(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.SimplifyRedundantBytesTrimSpace{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "bytes"

			func f(b []byte) []byte {
				return bytes.TrimSpace(b)
			}
		`),
	)
}
