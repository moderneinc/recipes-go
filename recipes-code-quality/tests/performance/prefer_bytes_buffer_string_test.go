/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package performance_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes-code-quality/recipes/performance"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestPreferBytesBufferStringSimple(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&performance.PreferBytesBufferString{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "bytes"

			func f(buf bytes.Buffer) string {
				return string(buf.Bytes())
			}
		`, `
			package main

			import "bytes"

			func f(buf bytes.Buffer) string {
				return buf.String()
			}
		`),
	)
}

func TestPreferBytesBufferStringNoChangeOtherMethod(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&performance.PreferBytesBufferString{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(buf interface{ Len() int }) int {
				return buf.Len()
			}
		`),
	)
}

func TestPreferBytesBufferStringNoChangePlainString(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&performance.PreferBytesBufferString{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(b []byte) string {
				return string(b)
			}
		`),
	)
}
