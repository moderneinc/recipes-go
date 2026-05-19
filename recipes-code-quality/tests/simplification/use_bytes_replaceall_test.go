/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes-code-quality/recipes/simplification"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestUseBytesReplaceAll(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.UseBytesReplaceAll{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "bytes"

			func f(b []byte) []byte {
				return bytes.Replace(b, []byte("old"), []byte("new"), -1)
			}
		`, `
			package main

			import "bytes"

			func f(b []byte) []byte {
				return bytes.ReplaceAll(b, []byte("old"), []byte("new"))
			}
		`),
	)
}

func TestUseBytesReplaceAllNoChange(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.UseBytesReplaceAll{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "bytes"

			func f(b []byte) []byte {
				return bytes.Replace(b, []byte("old"), []byte("new"), 1)
			}
		`),
	)
}
