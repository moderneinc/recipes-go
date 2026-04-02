/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/simplification"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestPreferStringsNewReader(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.PreferStringsNewReader{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "bytes"

			func f(s string) *bytes.Reader {
				return bytes.NewReader([]byte(s))
			}
		`, `
			package main

			import "bytes"

			func f(s string) *bytes.Reader {
				return strings.NewReader(s)
			}
		`),
	)
}

func TestPreferStringsNewReaderNoChange(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.PreferStringsNewReader{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "bytes"

			func f(b []byte) *bytes.Reader {
				return bytes.NewReader(b)
			}
		`),
	)
}
