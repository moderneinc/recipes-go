/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/simplification"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestSimplifyBytesSplitN(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.SimplifyBytesSplitN{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "bytes"

			func f(b []byte) [][]byte {
				return bytes.SplitN(b, []byte(","), -1)
			}
		`, `
			package main

			import "bytes"

			func f(b []byte) [][]byte {
				return bytes.Split(b, []byte(","))
			}
		`),
	)
}

func TestSimplifyBytesSplitNNoChange(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.SimplifyBytesSplitN{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "bytes"

			func f(b []byte) [][]byte {
				return bytes.SplitN(b, []byte(","), 2)
			}
		`),
	)
}
