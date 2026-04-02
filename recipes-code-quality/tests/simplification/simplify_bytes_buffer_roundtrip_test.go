/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/simplification"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestSimplifyBytesBufferRoundtrip(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.SimplifyBytesBufferRoundtrip{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "bytes"

			func f(b []byte) []byte {
				return bytes.NewBuffer(b).Bytes()
			}
		`, `
			package main

			import "bytes"

			func f(b []byte) []byte {
				return b
			}
		`),
	)
}

func TestSimplifyBytesBufferRoundtripNoChange(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.SimplifyBytesBufferRoundtrip{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "bytes"

			func f(b []byte) *bytes.Buffer {
				return bytes.NewBuffer(b)
			}
		`),
	)
}
